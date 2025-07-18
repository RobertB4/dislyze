package ip_whitelist

import (
	"encoding/json"
	"net/http"
	"strings"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/queries"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateLabelRequest struct {
	Label *string `json:"label"`
}

func (r *UpdateLabelRequest) Validate() error {
	if r.Label != nil {
		trimmed := strings.TrimSpace(*r.Label)
		r.Label = &trimmed
		
		if len(*r.Label) > 255 {
			return errlib.New(nil, http.StatusBadRequest, "")
		}
	}

	return nil
}

func (h *IPWhitelistHandler) UpdateIPLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := libctx.GetTenantID(ctx)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		appErr := errlib.New(nil, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var id pgtype.UUID
	if err := id.Scan(idStr); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var req UpdateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errlib.New(err, http.StatusBadRequest, "")
		responder.RespondWithError(w, appErr)
		return
	}

	if err := req.Validate(); err != nil {
		responder.RespondWithError(w, err)
		return
	}

	_, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			appErr := errlib.New(err, http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	var label pgtype.Text
	if req.Label != nil && *req.Label != "" {
		label = pgtype.Text{String: *req.Label, Valid: true}
	}

	err = h.q.UpdateIPWhitelistLabel(ctx, &queries.UpdateIPWhitelistLabelParams{
		Label:    label,
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		appErr := errlib.New(err, http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}
