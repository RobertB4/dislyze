package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/lib/permissions"
	"lugia/lib/responder"
	"lugia/queries"
)

type MeResponse struct {
	TenantName     string                     `json:"tenant_name"`
	UserID         string                     `json:"user_id"`
	Email          string                     `json:"email"`
	UserName       string                     `json:"user_name"`
	Permissions    []string                   `json:"permissions"`
	FeaturesConfig permissions.FeaturesConfig `json:"features_config"`
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

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusUnauthorized, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.New(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusUnauthorized, "")
		}
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	permissionRows, err := h.q.GetUserPermissions(ctx, &queries.GetUserPermissionsParams{
		UserID:   userID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, errlib.New(fmt.Errorf("GetMe: failed to get user permissions for user %s: %w", userID.String(), err), http.StatusInternalServerError, "")
	}

	permissionsRes := make([]string, len(permissionRows))
	for i, row := range permissionRows {
		permissionsRes[i] = fmt.Sprintf("%s.%s", row.Resource, row.Action)
	}

	var featuresConfig permissions.FeaturesConfig
	if err := json.Unmarshal(tenant.FeaturesConfig, &featuresConfig); err != nil {
		return nil, errlib.New(fmt.Errorf("GetMe: failed to unmarshal features config for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError, "")
	}

	response := &MeResponse{
		TenantName:     tenant.Name,
		UserID:         user.ID.String(),
		Email:          user.Email,
		UserName:       user.Name,
		Permissions:    permissionsRes,
		FeaturesConfig: featuresConfig,
	}

	return response, nil
}
