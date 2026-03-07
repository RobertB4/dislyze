// Feature doc: docs/features/rbac.md
package roles

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

var CreateRoleOp = huma.Operation{
	OperationID: "create-role",
	Method:      http.MethodPost,
	Path:        "/roles/create",
}

type CreateRoleInput struct {
	Body CreateRoleRequestBody
}

type CreateRoleRequestBody struct {
	Name          string   `json:"name" minLength:"1"`
	Description   string   `json:"description"`
	PermissionIDs []string `json:"permission_ids" minItems:"1"`
}

func (h *RolesHandler) CreateRole(ctx context.Context, input *CreateRoleInput) (*struct{}, error) {
	err := h.createRole(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *RolesHandler) createRole(ctx context.Context, req CreateRoleRequestBody) error {
	tenantID := libctx.GetTenantID(ctx)

	permissionIDs := make([]pgtype.UUID, len(req.PermissionIDs))
	for i, permissionIDStr := range req.PermissionIDs {
		var permissionID pgtype.UUID
		err := permissionID.Scan(permissionIDStr)
		if err != nil {
			return errlib.NewError(fmt.Errorf("CreateRole: invalid permission ID format %s: %w", permissionIDStr, err), http.StatusInternalServerError)
		}
		permissionIDs[i] = permissionID
	}

	allPermissions, err := h.q.GetAllPermissions(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("CreateRole: failed to get all permissions: %w", err), http.StatusInternalServerError)
	}

	permissionMap := make(map[pgtype.UUID]bool)
	for _, permission := range allPermissions {
		permissionMap[permission.ID] = true
	}

	for _, permissionID := range permissionIDs {
		if !permissionMap[permissionID] {
			return errlib.NewError(fmt.Errorf("CreateRole: permission ID %s does not exist", permissionID.String()), http.StatusInternalServerError)
		}
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("CreateRole: failed to begin transaction: %w", err), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("CreateRole: failed to create role: %w", err), http.StatusInternalServerError)
	}

	err = qtx.CreateRolePermissionsBulk(ctx, &queries.CreateRolePermissionsBulkParams{
		RoleID:        createdRole.ID,
		PermissionIds: permissionIDs,
		TenantID:      tenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("CreateRole: failed to assign permissions to role: %w", err), http.StatusInternalServerError)
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("CreateRole: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
