package roles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/queries"
)

type UpdateRoleRequestBody struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

func (r *UpdateRoleRequestBody) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)

	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

func (h *RolesHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	roleIDStr := r.PathValue("roleID")
	if roleIDStr == "" {
		appErr := errlib.New(fmt.Errorf("UpdateRole: roleID is required"), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var roleID pgtype.UUID
	if err := roleID.Scan(roleIDStr); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateRole: invalid role ID format %s: %w", roleIDStr, err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var req UpdateRoleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateRole: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("UpdateRole: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("UpdateRole: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.updateRole(ctx, roleID, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *RolesHandler) updateRole(ctx context.Context, roleID pgtype.UUID, req UpdateRoleRequestBody) error {
	tenantID := libctx.GetTenantID(ctx)

	role, err := h.q.GetRoleByID(ctx, &queries.GetRoleByIDParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("UpdateRole: role not found"), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("UpdateRole: failed to get role: %w", err), http.StatusInternalServerError, "")
	}

	if role.IsDefault {
		return errlib.New(fmt.Errorf("UpdateRole: cannot update default role"), http.StatusBadRequest, "")
	}

	permissionIDs := make([]pgtype.UUID, len(req.PermissionIDs))
	for i, permissionIDStr := range req.PermissionIDs {
		var permissionID pgtype.UUID
		err := permissionID.Scan(permissionIDStr)
		if err != nil {
			return errlib.New(fmt.Errorf("UpdateRole: invalid permission ID format %s: %w", permissionIDStr, err), http.StatusInternalServerError, "")
		}
		permissionIDs[i] = permissionID
	}

	if len(permissionIDs) > 0 {
		allPermissions, err := h.q.GetAllPermissions(ctx)
		if err != nil {
			return errlib.New(fmt.Errorf("UpdateRole: failed to get all permissions: %w", err), http.StatusInternalServerError, "")
		}

		permissionMap := make(map[pgtype.UUID]bool)
		for _, permission := range allPermissions {
			permissionMap[permission.ID] = true
		}

		for _, permissionID := range permissionIDs {
			if !permissionMap[permissionID] {
				return errlib.New(fmt.Errorf("UpdateRole: permission ID %s does not exist", permissionID.String()), http.StatusInternalServerError, "")
			}
		}
	}

	if req.Name != role.Name {
		exists, err := h.q.CheckRoleNameExists(ctx, &queries.CheckRoleNameExistsParams{
			TenantID: tenantID,
			Name:     req.Name,
			ID:       roleID,
		})
		if err != nil {
			return errlib.New(fmt.Errorf("UpdateRole: failed to check role name exists: %w", err), http.StatusInternalServerError, "")
		}
		if exists {
			return errlib.New(fmt.Errorf("UpdateRole: role name already exists"), http.StatusBadRequest, "")
		}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("UpdateRole: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("UpdateRole: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	description := pgtype.Text{String: req.Description, Valid: req.Description != ""}
	err = qtx.UpdateRole(ctx, &queries.UpdateRoleParams{
		Name:        req.Name,
		Description: description,
		ID:          roleID,
		TenantID:    tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("UpdateRole: failed to update role: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.DeleteRolePermissions(ctx, &queries.DeleteRolePermissionsParams{
		RoleID:   roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("UpdateRole: failed to delete existing permissions: %w", err), http.StatusInternalServerError, "")
	}

	if len(permissionIDs) > 0 {
		err = qtx.CreateRolePermissionsBulk(ctx, &queries.CreateRolePermissionsBulkParams{
			RoleID:        roleID,
			PermissionIds: permissionIDs,
			TenantID:      tenantID,
		})
		if err != nil {
			return errlib.New(fmt.Errorf("UpdateRole: failed to assign permissions to role: %w", err), http.StatusInternalServerError, "")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("UpdateRole: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
