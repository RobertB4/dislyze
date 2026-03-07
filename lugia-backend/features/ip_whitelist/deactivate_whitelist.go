// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
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

	return nil
}
