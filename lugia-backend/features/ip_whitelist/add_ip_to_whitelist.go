// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
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
	IPAddress string  `json:"ip_address"`
	Label     *string `json:"label"`
}

func (r *AddIPToWhitelistRequest) Validate() error {
	if r.IPAddress == "" {
		return errlib.New(nil, http.StatusBadRequest, "")
	}

	_, err := iputils.ValidateCIDR(r.IPAddress)
	if err != nil {
		return errlib.New(err, http.StatusBadRequest, "")
	}

	if r.Label != nil && len(*r.Label) > 255 {
		return errlib.New(nil, http.StatusBadRequest, "")
	}

	return nil
}

func (h *IPWhitelistHandler) AddIPToWhitelist(ctx context.Context, input *AddIPInput) (*struct{}, error) {
	if err := input.Body.Validate(); err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusBadRequest)
	}

	err := h.addIPToWhitelist(ctx, input.Body)
	if err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			if appErr.Message != "" {
				return nil, humautil.NewErrorWithDetail(err, appErr.StatusCode, appErr.Message)
			}
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}
	return nil, nil
}

func (h *IPWhitelistHandler) addIPToWhitelist(ctx context.Context, req AddIPToWhitelistRequest) error {
	tenantID := libctx.GetTenantID(ctx)
	userID := libctx.GetUserID(ctx)

	normalizedCIDR, err := iputils.ValidateCIDR(req.IPAddress)
	if err != nil {
		return errlib.New(err, http.StatusBadRequest, "")
	}

	prefix, err := netip.ParsePrefix(normalizedCIDR)
	if err != nil {
		return errlib.New(err, http.StatusBadRequest, "")
	}

	exists, err := h.q.CheckIPExists(ctx, &queries.CheckIPExistsParams{
		TenantID:  tenantID,
		IpAddress: prefix,
	})
	if err != nil {
		return errlib.New(err, http.StatusInternalServerError, "")
	}
	if exists {
		return errlib.New(nil, http.StatusBadRequest, "")
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
		return errlib.New(err, http.StatusInternalServerError, "")
	}

	return nil
}
