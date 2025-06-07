package users

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
)

type MeResponse struct {
	TenantName string `json:"tenant_name"`
	TenantPlan string `json:"tenant_plan"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	UserName   string `json:"user_name"`
	UserRole   string `json:"user_role"`
}

func (h *UsersHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			appErr := errlib.New(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusNotFound, "")
			responder.RespondWithError(w, appErr)
			return
		}
		appErr := errlib.New(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
		responder.RespondWithError(w, appErr)
		return
	}

	response := MeResponse{
		TenantName: tenant.Name,
		TenantPlan: tenant.Plan,
		UserID:     user.ID.String(),
		Email:      user.Email,
		UserName:   user.Name,
		UserRole:   user.Role.String(),
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}