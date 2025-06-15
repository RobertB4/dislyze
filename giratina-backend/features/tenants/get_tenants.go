package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"dislyze/jirachi/authz"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/responder"
)

type TenantResponse struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	EnterpriseFeatures authz.EnterpriseFeatures `json:"enterprise_features"`
	StripeCustomerID   string                   `json:"stripe_customer_id,omitempty"`
	CreatedAt          string                   `json:"created_at"`
	UpdatedAt          string                   `json:"updated_at"`
}

type GetTenantsResponse struct {
	Tenants []TenantResponse `json:"tenants"`
}

func (h *TenantsHandler) GetTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenants, err := h.getTenants(ctx)
	if err != nil {
		responder.RespondWithError(w, err)
		return
	}

	response := GetTenantsResponse{
		Tenants: tenants,
	}

	responder.RespondWithJSON(w, http.StatusOK, response)
}

func (h *TenantsHandler) getTenants(ctx context.Context) ([]TenantResponse, error) {
	dbTenants, err := h.queries.GetTenants(ctx)
	if err != nil {
		return nil, errlib.New(fmt.Errorf("GetTenants: failed to get tenants: %w", err), http.StatusInternalServerError, "")
	}

	tenants := make([]TenantResponse, len(dbTenants))
	for i, tenant := range dbTenants {

		enterpriseFeatures := authz.EnterpriseFeatures{}
		if len(tenant.EnterpriseFeatures) > 0 {
			if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
				return nil, errlib.New(fmt.Errorf("GetMe: failed to unmarshal features config for tenant %s: %w", tenant.ID.String(), err), http.StatusInternalServerError, "")
			}
		}

		stripeCustomerID := ""
		if tenant.StripeCustomerID.Valid {
			stripeCustomerID = tenant.StripeCustomerID.String
		}

		tenants[i] = TenantResponse{
			ID:                 tenant.ID.String(),
			Name:               tenant.Name,
			EnterpriseFeatures: enterpriseFeatures,
			StripeCustomerID:   stripeCustomerID,
			CreatedAt:          tenant.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          tenant.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return tenants, nil
}
