// Feature doc: docs/features/rbac.md, docs/features/ip-whitelisting.md, docs/features/audit-logging.md
package tenants

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"giratina/lib/middleware"
	"giratina/queries"
	"net"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	"dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
)

var UpdateTenantOp = huma.Operation{
	OperationID: "update-tenant",
	Method:      http.MethodPost,
	Path:        "/tenants/{id}/update",
}

type UpdateTenantInput struct {
	ID   string `path:"id"`
	Body UpdateTenantRequestBody
}

type UpdateTenantRequestBody struct {
	Name               string                   `json:"name" minLength:"1"`
	EnterpriseFeatures authz.EnterpriseFeatures `json:"enterprise_features"`
}

func (h *TenantsHandler) UpdateTenant(ctx context.Context, input *UpdateTenantInput) (*struct{}, error) {
	var tenantID pgtype.UUID
	if err := tenantID.Scan(input.ID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid tenant ID format: %w", err), http.StatusBadRequest)
	}

	if err := h.updateTenant(ctx, &tenantID, &input.Body); err != nil {
		return nil, err
	}

	return nil, nil
}

func (h *TenantsHandler) updateTenant(ctx context.Context, tenantID *pgtype.UUID, requestBody *UpdateTenantRequestBody) error {
	enterpriseFeaturesJSON, err := json.Marshal(requestBody.EnterpriseFeatures)
	if err != nil {
		return errlib.NewError(fmt.Errorf("failed to marshal enterprise features: %w", err), http.StatusInternalServerError)
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.NewError(fmt.Errorf("UpdateTenant: failed to begin transaction: %w", err), http.StatusInternalServerError)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("UpdateTenant: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.queries.WithTx(tx)

	err = qtx.UpdateTenant(ctx, &queries.UpdateTenantParams{
		Name:               requestBody.Name,
		EnterpriseFeatures: enterpriseFeaturesJSON,
		ID:                 *tenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("failed to update tenant: %w", err), http.StatusInternalServerError)
	}

	// Check if target tenant has audit logging enabled (in the NEW features being set)
	if requestBody.EnterpriseFeatures.AuditLog.Enabled {
		r := middleware.GetHTTPRequest(ctx)
		userID := libctx.GetUserID(ctx)

		metadata, _ := json.Marshal(map[string]string{
			"is_internal_admin":    "true",
			"enterprise_features":  string(enterpriseFeaturesJSON),
		})

		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ipAddr, _ := netip.ParseAddr(host)
		err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     *tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceTenant),
			Action:       string(auditlog.ActionEnterpriseFeatureToggled),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("UpdateTenant: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.NewError(fmt.Errorf("UpdateTenant: failed to commit transaction: %w", err), http.StatusInternalServerError)
	}

	return nil
}
