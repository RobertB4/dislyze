package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/queries"
)

func (h *IPWhitelistHandler) DeactivateWhitelist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.rateLimiter.Allow(libctx.GetUserID(ctx).String(), r) {
		appErr := errlib.New(fmt.Errorf("DeactivateWhitelist: rate limit exceeded"), http.StatusTooManyRequests, "")
		responder.RespondWithError(w, appErr)
		return
	}

	err := h.deactivateWhitelist(ctx)
	if err != nil {
		appErr := errlib.New(fmt.Errorf("DeactivateWhitelist: %w", err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
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
