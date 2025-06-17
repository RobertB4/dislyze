package tenants

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
)

type UserInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type GetUsersByTenantResponse struct {
	Users []UserInfo `json:"users"`
}

func (h *TenantsHandler) GetUsersByTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "tenantID")

	var tenantID pgtype.UUID
	if err := tenantID.Scan(tenantIDStr); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("GetUsersByTenant: invalid tenant ID: %w", err), http.StatusBadRequest, ""))
		return
	}

	users, err := h.getUsersByTenant(ctx, tenantID)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	response := GetUsersByTenantResponse{
		Users: users,
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *TenantsHandler) getUsersByTenant(ctx context.Context, tenantID pgtype.UUID) ([]UserInfo, error) {
	dbUsers, err := h.queries.GetUsersByTenantID(ctx, tenantID)
	if err != nil {
		return nil, errlib.New(fmt.Errorf("GetUsersByTenant: failed to get users by tenant: %w", err), http.StatusInternalServerError, "")
	}

	users := make([]UserInfo, len(dbUsers))
	for i, user := range dbUsers {
		users[i] = UserInfo{
			ID:     user.ID.String(),
			Name:   user.Name,
			Email:  user.Email,
			Status: user.Status,
		}
	}

	return users, nil
}
