// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var DeleteIPOp = huma.Operation{
	OperationID: "delete-ip",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/{id}/delete",
}

type DeleteIPInput struct {
	ID string `path:"id"`
}

func (h *IPWhitelistHandler) DeleteIP(ctx context.Context, input *DeleteIPInput) (*struct{}, error) {
	var id pgtype.UUID
	if err := id.Scan(input.ID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid IP whitelist rule ID format: %w", err), http.StatusBadRequest)
	}

	err := h.deleteIP(ctx, id)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) deleteIP(ctx context.Context, id pgtype.UUID) error {
	tenantID := libctx.GetTenantID(ctx)

	rule, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return errlib.NewError(err, http.StatusNotFound)
		}
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	ipConfig := libctx.GetIPWhitelistConfig(ctx)
	if ipConfig.Active {
		r := middleware.GetHTTPRequest(ctx)
		clientIP := iputils.ExtractClientIP(r)

		isCurrentIP, err := iputils.IsIPInCIDRList(clientIP, []string{rule.IpAddress.String()})
		if err != nil {
			return errlib.NewError(err, http.StatusInternalServerError)
		}

		if isCurrentIP {
			return errlib.NewErrorWithDetail(nil, http.StatusBadRequest, "現在使用中のIPアドレスは削除できません。")
		}
	}

	err = h.q.RemoveIPFromWhitelist(ctx, &queries.RemoveIPFromWhitelistParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	return nil
}
