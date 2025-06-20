package ip_whitelist

import (
	"encoding/json"
	"net/http"
	"net/netip"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/lib/iputils"
	"lugia/queries"

	"github.com/jackc/pgx/v5/pgtype"
)

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

	return nil
}

func (h *IPWhitelistHandler) AddIPToWhitelist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := libctx.GetTenantID(ctx)
	userID := libctx.GetUserID(ctx)

	var req AddIPToWhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := req.Validate(); err != nil {
		responder.RespondWithError(w, err)
		return
	}

	normalizedCIDR, err := iputils.ValidateCIDR(req.IPAddress)
	if err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	prefix, err := netip.ParsePrefix(normalizedCIDR)
	if err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
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
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}
