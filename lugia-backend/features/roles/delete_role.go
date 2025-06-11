package roles

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
	"lugia/queries"
)

func (h *RolesHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	roleIDStr := r.PathValue("roleID")
	if roleIDStr == "" {
		appErr := errlib.New(fmt.Errorf("DeleteRole: roleID is required"), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var roleID pgtype.UUID
	if err := roleID.Scan(roleIDStr); err != nil {
		appErr := errlib.New(fmt.Errorf("DeleteRole: invalid role ID format %s: %w", roleIDStr, err), http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.deleteRole(ctx, roleID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *RolesHandler) deleteRole(ctx context.Context, roleID pgtype.UUID) error {
	tenantID := libctx.GetTenantID(ctx)

	role, err := h.q.GetRoleByID(ctx, &queries.GetRoleByIDParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("DeleteRole: role not found"), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("DeleteRole: failed to get role: %w", err), http.StatusInternalServerError, "")
	}

	if role.IsDefault {
		return errlib.New(fmt.Errorf("DeleteRole: cannot delete default role"), http.StatusBadRequest, "")
	}

	inUse, err := h.q.CheckRoleInUse(ctx, &queries.CheckRoleInUseParams{
		RoleID:   roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("DeleteRole: failed to check if role is in use: %w", err), http.StatusInternalServerError, "")
	}

	if inUse {
		return errlib.New(fmt.Errorf("DeleteRole: role is assigned to users"), http.StatusBadRequest, "このロールはユーザーに割り当てられているため削除できません。")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("DeleteRole: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("DeleteRole: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.DeleteRolePermissions(ctx, &queries.DeleteRolePermissionsParams{
		RoleID:   roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("DeleteRole: failed to delete role permissions: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.DeleteRole(ctx, &queries.DeleteRoleParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("DeleteRole: failed to delete role: %w", err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("DeleteRole: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
