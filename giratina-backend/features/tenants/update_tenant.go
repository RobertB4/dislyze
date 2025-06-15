package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"giratina/queries"
	"net/http"
	"strings"

	"dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateTenantRequestBody struct {
	Name               string                   `json:"name"`
	EnterpriseFeatures authz.EnterpriseFeatures `json:"enterprise_features"`
}

func (r *UpdateTenantRequestBody) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type UpdateTenantResponse struct {
	Message string `json:"message"`
}

func (h *TenantsHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantIDStr := chi.URLParam(r, "id")
	if tenantIDStr == "" {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("tenant ID is required"), http.StatusBadRequest, ""))
		return
	}

	var tenantID pgtype.UUID
	if err := tenantID.Scan(tenantIDStr); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("invalid tenant ID format"), http.StatusBadRequest, ""))
		return
	}

	var requestBody UpdateTenantRequestBody
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		responder.RespondWithError(w, errlib.New(fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest, ""))
		return
	}

	if err := requestBody.Validate(); err != nil {
		responder.RespondWithError(w, errlib.New(err, http.StatusBadRequest, ""))
		return
	}

	if err := h.updateTenant(ctx, &tenantID, &requestBody); err != nil {
		responder.RespondWithError(w, err)
		return
	}

	response := UpdateTenantResponse{
		Message: "Tenant updated successfully",
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *TenantsHandler) updateTenant(ctx context.Context, tenantID *pgtype.UUID, requestBody *UpdateTenantRequestBody) error {
	enterpriseFeaturesJSON, err := json.Marshal(requestBody.EnterpriseFeatures)
	if err != nil {
		return errlib.New(fmt.Errorf("failed to marshal enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	err = h.queries.UpdateTenant(ctx, &queries.UpdateTenantParams{
		Name:               requestBody.Name,
		EnterpriseFeatures: enterpriseFeaturesJSON,
		ID:                 *tenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("failed to update tenant: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}
