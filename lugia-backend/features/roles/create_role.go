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

type CreateRoleRequestBody struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

func (r *CreateRoleRequestBody) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)

	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(r.PermissionIDs) == 0 {
		return fmt.Errorf("at least one permission is required")
	}
	return nil
}

func (h *RolesHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRoleRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(fmt.Errorf("CreateRole: failed to decode request: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			errlib.LogError(fmt.Errorf("CreateRole: failed to close request body: %w", err))
		}
	}()

	if err := req.Validate(); err != nil {
		appErr := errlib.New(fmt.Errorf("CreateRole: validation failed: %w", err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.createRole(ctx, req)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *RolesHandler) createRole(ctx context.Context, req CreateRoleRequestBody) error {
	tenantID := libctx.GetTenantID(ctx)

	permissionIDs := make([]pgtype.UUID, len(req.PermissionIDs))
	for i, permissionIDStr := range req.PermissionIDs {
		var permissionID pgtype.UUID
		err := permissionID.Scan(permissionIDStr)
		if err != nil {
			return errlib.New(fmt.Errorf("CreateRole: invalid permission ID format %s: %w", permissionIDStr, err), http.StatusInternalServerError, "")
		}
		permissionIDs[i] = permissionID
	}

	allPermissions, err := h.q.GetAllPermissions(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("CreateRole: failed to get all permissions: %w", err), http.StatusInternalServerError, "")
	}

	permissionMap := make(map[pgtype.UUID]bool)
	for _, permission := range allPermissions {
		permissionMap[permission.ID] = true
	}

	for _, permissionID := range permissionIDs {
		if !permissionMap[permissionID] {
			return errlib.New(fmt.Errorf("CreateRole: permission ID %s does not exist", permissionID.String()), http.StatusInternalServerError, "")
		}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("CreateRole: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("CreateRole: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	description := pgtype.Text{String: req.Description, Valid: req.Description != ""}
	createdRole, err := qtx.CreateRole(ctx, &queries.CreateRoleParams{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: description,
		IsDefault:   false,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("CreateRole: failed to create role: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.CreateRolePermissionsBulk(ctx, &queries.CreateRolePermissionsBulkParams{
		RoleID:        createdRole.ID,
		PermissionIds: permissionIDs,
		TenantID:      tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("CreateRole: failed to assign permissions to role: %w", err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("CreateRole: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
