package ip_whitelist

import (
	"net/http"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
	"lugia/lib/iputils"
	"lugia/queries"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *IPWhitelistHandler) DeleteIP(w http.ResponseWriter, r *http.Request) {
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

	rule, err := h.q.GetIPWhitelistRuleByID(ctx, &queries.GetIPWhitelistRuleByIDParams{
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

	ipConfig := libctx.GetIPWhitelistConfig(ctx)
	if ipConfig.Active {
		clientIP := iputils.ExtractClientIP(r)

		isCurrentIP, err := iputils.IsIPInCIDRList(clientIP, []string{rule.IpAddress.String()})
		if err != nil {
			appErr := errlib.New(err, http.StatusInternalServerError, "")
			responder.RespondWithError(w, appErr)
			return
		}

		if isCurrentIP {
			appErr := errlib.New(nil, http.StatusBadRequest, "現在使用中のIPアドレスは削除できません。")
			responder.RespondWithError(w, appErr)
			return
		}
	}

	err = h.q.RemoveIPFromWhitelist(ctx, &queries.RemoveIPFromWhitelistParams{
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
