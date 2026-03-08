// Feature doc: docs/features/profile-management.md, docs/features/audit-logging.md
package users

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

var ChangeTenantNameOp = huma.Operation{
	OperationID: "change-tenant-name",
	Method:      http.MethodPost,
	Path:        "/tenant/change-name",
}

type ChangeTenantNameInput struct {
	Body ChangeTenantNameRequestBody
}

type ChangeTenantNameRequestBody struct {
	Name string `json:"name" minLength:"1"`
}

func (h *UsersHandler) ChangeTenantName(ctx context.Context, input *ChangeTenantNameInput) (*struct{}, error) {
	tenantID := libctx.GetTenantID(ctx)
	err := h.changeTenantName(ctx, tenantID, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) changeTenantName(ctx context.Context, tenantID pgtype.UUID, req ChangeTenantNameRequestBody) error {
	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("ChangeTenantName: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	// Fetch old name before update for audit log metadata.
	var oldName string
	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		tenant, err := qtx.GetTenantByID(ctx, tenantID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to get tenant for audit log: %w", err), http.StatusInternalServerError)
		}
		oldName = tenant.Name
	}

	if err := qtx.UpdateTenantName(ctx, &queries.UpdateTenantNameParams{
		Name: req.Name,
		ID:   tenantID,
	}); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to update tenant name for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		actorUserID := libctx.GetUserID(ctx)
		actorDBUser, err := qtx.GetUserByID(ctx, actorUserID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to get actor user details for audit log: %w", err), http.StatusInternalServerError)
		}

		r := middleware.GetHTTPRequest(ctx)
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actorDBUser.Name,
			"actor_email": actorDBUser.Email,
			"old_name":    oldName,
			"new_name":    req.Name,
		})

		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      actorUserID,
			ResourceType: string(auditlog.ResourceTenant),
			Action:       string(auditlog.ActionNameChanged),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{String: tenantID.String(), Valid: true},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("ChangeTenantName: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
