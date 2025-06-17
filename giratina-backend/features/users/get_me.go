package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
)

type MeResponse struct {
	TenantName string `json:"tenant_name"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	UserName   string `json:"user_name"`
}

func (h *UsersHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response, err := h.getMe(ctx)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *UsersHandler) getMe(ctx context.Context) (*MeResponse, error) {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	user, err := h.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusUnauthorized, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	tenant, err := h.queries.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusUnauthorized, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	response := &MeResponse{
		TenantName: tenant.Name,
		UserID:     user.ID.String(),
		Email:      user.Email,
		UserName:   user.Name,
	}

	return response, nil
}
