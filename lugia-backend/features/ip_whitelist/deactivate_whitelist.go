// Feature doc: docs/features/ip-whitelisting.md, docs/features/audit-logging.md
package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var DeactivateWhitelistOp = huma.Operation{
	OperationID: "deactivate-whitelist",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/deactivate",
}

type DeactivateWhitelistInput struct{}

func (h *IPWhitelistHandler) DeactivateWhitelist(ctx context.Context, input *DeactivateWhitelistInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)

	if !h.rateLimiter.Allow(libctx.GetUserID(ctx).String(), r) {
		return nil, errlib.NewError(fmt.Errorf("rate limit exceeded for deactivate whitelist"), http.StatusTooManyRequests)
	}

	err := h.deactivateWhitelist(ctx)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("DeactivateWhitelist: %w", err), http.StatusInternalServerError)
	}

	return nil, nil
}

func (h *IPWhitelistHandler) deactivateWhitelist(ctx context.Context) error {
	tenantID := libctx.GetTenantID(ctx)
	enterpriseFeatures := libctx.GetEnterpriseFeatures(ctx)

	enterpriseFeatures.IPWhitelist.Active = false

	updatedFeaturesJSON, err := json.Marshal(enterpriseFeatures)
	if err != nil {
		return fmt.Errorf("failed to marshal enterprise features: %w", err)
	}

	err = h.q.UpdateTenantEnterpriseFeatures(ctx, &queries.UpdateTenantEnterpriseFeaturesParams{
		EnterpriseFeatures: updatedFeaturesJSON,
		ID:                 tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to update tenant enterprise features: %w", err)
	}

	if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
		r := middleware.GetHTTPRequest(ctx)
		userID := libctx.GetUserID(ctx)
		actor, err := h.q.GetUserByID(ctx, userID)
		if err != nil {
			return errlib.NewError(fmt.Errorf("DeactivateWhitelist: failed to get actor for audit log: %w", err), http.StatusInternalServerError)
		}
		metadata, _ := json.Marshal(map[string]string{
			"actor_name":  actor.Name,
			"actor_email": actor.Email,
		})
		ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
		err = h.q.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
			TenantID:     tenantID,
			ActorID:      userID,
			ResourceType: string(auditlog.ResourceIPWhitelist),
			Action:       string(auditlog.ActionDeactivated),
			Outcome:      string(auditlog.OutcomeSuccess),
			ResourceID:   pgtype.Text{},
			Metadata:     metadata,
			IpAddress:    &ipAddr,
			UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
		})
		if err != nil {
			return errlib.NewError(fmt.Errorf("DeactivateWhitelist: failed to insert audit log: %w", err), http.StatusInternalServerError)
		}
	}

	return nil
}
