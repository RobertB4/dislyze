// Feature doc: docs/features/rbac.md, docs/features/audit-logging.md
package roles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var DeleteRoleOp = huma.Operation{
	OperationID: "delete-role",
	Method:      http.MethodPost,
	Path:        "/roles/{roleID}/delete",
}

type DeleteRoleInput struct {
	RoleID string `path:"roleID"`
}

func (h *RolesHandler) DeleteRole(ctx context.Context, input *DeleteRoleInput) (*struct{}, error) {
	var roleID pgtype.UUID
	if err := roleID.Scan(input.RoleID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid role ID format for delete: %w", err), http.StatusBadRequest)
	}

	err := h.deleteRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *RolesHandler) deleteRole(ctx context.Context, roleID pgtype.UUID) error {
	tenantID := libctx.GetTenantID(ctx)

	role, err := h.q.GetRoleByID(ctx, &queries.GetRoleByIDParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.NewError(fmt.Errorf("DeleteRole: role not found"), http.StatusNotFound)
		}
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to get role: %w", err), http.StatusInternalServerError)
	}

	if role.IsDefault {
		return errlib.NewError(fmt.Errorf("DeleteRole: cannot delete default role"), http.StatusBadRequest)
	}

	inUse, err := h.q.CheckRoleInUse(ctx, &queries.CheckRoleInUseParams{
		RoleID:   roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to check if role is in use: %w", err), http.StatusInternalServerError)
	}

	if inUse {
		return errlib.NewErrorWithDetail(fmt.Errorf("DeleteRole: role is assigned to users"), http.StatusBadRequest, "このロールはユーザーに割り当てられているため削除できません。")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to begin transaction: %w", err), http.StatusInternalServerError)
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
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to delete role permissions: %w", err), http.StatusInternalServerError)
	}

	err = qtx.DeleteRole(ctx, &queries.DeleteRoleParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to delete role: %w", err), http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		r := middleware.GetHTTPRequest(ctx)
		actor, err := qtx.GetUserByID(ctx, libctx.GetUserID(ctx))
		if err != nil {
			return errlib.NewError(fmt.Errorf("DeleteRole: failed to get actor for audit log: %w", err), http.StatusInternalServerError)
		}

		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actor.Name,
			"actor_email": actor.Email,
			"role_name":   role.Name,
		})

		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      actor.ID,
			ResourceType: string(auditlog.ResourceRole),
			Action:       string(auditlog.ActionDeleted),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: roleID.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("DeleteRole: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("DeleteRole: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
