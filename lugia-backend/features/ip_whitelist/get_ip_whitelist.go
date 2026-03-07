// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
)

var GetIPWhitelistOp = huma.Operation{
	OperationID: "get-ip-whitelist",
	Method:      http.MethodGet,
	Path:        "/ip-whitelist",
}

type IPWhitelistRule struct {
	ID        string    `json:"id"`
	IPAddress string    `json:"ip_address"`
	Label     *string   `json:"label"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type GetIPWhitelistInput struct{}

type GetIPWhitelistResponse struct {
	Rules []IPWhitelistRule `json:"rules" nullable:"false"`
}

type GetIPWhitelistOutput struct {
	Body GetIPWhitelistResponse
}

func (h *IPWhitelistHandler) GetIPWhitelist(ctx context.Context, input *GetIPWhitelistInput) (*GetIPWhitelistOutput, error) {
	tenantID := libctx.GetTenantID(ctx)

	ipRules, err := h.q.GetTenantIPWhitelist(ctx, tenantID)
	if err != nil {
		return nil, errlib.NewError(err, http.StatusInternalServerError)
	}

	rules := make([]IPWhitelistRule, len(ipRules))
	for i, rule := range ipRules {
		var label *string
		if rule.Label.Valid {
			label = &rule.Label.String
		}

		rules[i] = IPWhitelistRule{
			ID:        rule.ID.String(),
			IPAddress: rule.IpAddress.String(),
			Label:     label,
			CreatedBy: rule.CreatedBy.String(),
			CreatedAt: rule.CreatedAt.Time,
		}
	}

	return &GetIPWhitelistOutput{Body: GetIPWhitelistResponse{Rules: rules}}, nil
}
