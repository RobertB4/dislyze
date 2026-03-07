// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/iputils"
	"lugia/queries"
)

var AddIPOp = huma.Operation{
	OperationID: "add-ip-to-whitelist",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/create",
}

type AddIPInput struct {
	Body AddIPToWhitelistRequest
}

type AddIPToWhitelistRequest struct {
	IPAddress string  `json:"ip_address" minLength:"1"`
	Label     *string `json:"label" maxLength:"255"`
}

func (r *AddIPToWhitelistRequest) Resolve(ctx huma.Context) []error {
	if _, err := iputils.ValidateCIDR(r.IPAddress); err != nil {
		return []error{fmt.Errorf("invalid IP address or CIDR: %w", err)}
	}
	return nil
}

func (h *IPWhitelistHandler) AddIPToWhitelist(ctx context.Context, input *AddIPInput) (*struct{}, error) {
	err := h.addIPToWhitelist(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) addIPToWhitelist(ctx context.Context, req AddIPToWhitelistRequest) error {
	tenantID := libctx.GetTenantID(ctx)
	userID := libctx.GetUserID(ctx)

	normalizedCIDR, err := iputils.ValidateCIDR(req.IPAddress)
	if err != nil {
		return errlib.NewError(err, http.StatusBadRequest)
	}

	prefix, err := netip.ParsePrefix(normalizedCIDR)
	if err != nil {
		return errlib.NewError(err, http.StatusBadRequest)
	}

	exists, err := h.q.CheckIPExists(ctx, &queries.CheckIPExistsParams{
		TenantID:  tenantID,
		IpAddress: prefix,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}
	if exists {
		return errlib.NewError(nil, http.StatusBadRequest)
	}

	var label pgtype.Text
	if req.Label != nil {
		label = pgtype.Text{String: *req.Label, Valid: true}
	}

	_, err = h.q.AddIPToWhitelist(ctx, &queries.AddIPToWhitelistParams{
		TenantID:  tenantID,
		IpAddress: prefix,
		Label:     label,
		CreatedBy: userID,
	})
	if err != nil {
		return errlib.NewError(err, http.StatusInternalServerError)
	}

	return nil
}
