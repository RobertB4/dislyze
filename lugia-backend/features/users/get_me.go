package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/responder"
)

func (h *UsersHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	response, err := h.getMe(ctx, userID, tenantID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *UsersHandler) getMe(ctx context.Context, userID, tenantID pgtype.UUID) (*MeResponse, error) {
	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusNotFound, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusNotFound, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	response := &MeResponse{
		TenantName: tenant.Name,
		TenantPlan: tenant.Plan,
		UserID:     user.ID.String(),
		Email:      user.Email,
		UserName:   user.Name,
		UserRole:   user.Role.String(),
	}

	return response, nil
}
