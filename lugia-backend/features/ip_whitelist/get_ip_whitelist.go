package ip_whitelist

import (
	"net/http"
	"time"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
)

type IPWhitelistRule struct {
	ID        string    `json:"id"`
	IPAddress string    `json:"ip_address"`
	Label     *string   `json:"label"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *IPWhitelistHandler) GetIPWhitelist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := libctx.GetTenantID(ctx)

	ipRules, err := h.q.GetTenantIPWhitelist(ctx, tenantID)
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	response := make([]IPWhitelistRule, len(ipRules))
	for i, rule := range ipRules {
		var label *string
		if rule.Label.Valid {
			label = &rule.Label.String
		}

		response[i] = IPWhitelistRule{
			ID:        rule.ID.String(),
			IPAddress: rule.IpAddress.String(),
			Label:     label,
			CreatedBy: rule.CreatedBy.String(),
			CreatedAt: rule.CreatedAt.Time,
		}
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}
