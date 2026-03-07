// Feature doc: docs/features/tenant-onboarding.md
package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"dislyze/jirachi/authz"
	"giratina/lib/humautil"
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

var GetTenantsOp = huma.Operation{
	OperationID: "get-tenants",
	Method:      http.MethodGet,
	Path:        "/tenants",
}

type GetTenantsInput struct{}

type GetTenantsOutput struct {
	Body GetTenantsResponse
}

func (h *TenantsHandler) GetTenants(ctx context.Context, input *GetTenantsInput) (*GetTenantsOutput, error) {
	tenants, err := h.getTenants(ctx)
	if err != nil {
		return nil, err
	}

	return &GetTenantsOutput{Body: GetTenantsResponse{Tenants: tenants}}, nil
}

func (h *TenantsHandler) getTenants(ctx context.Context) ([]TenantResponse, error) {
	dbTenants, err := h.queries.GetTenants(ctx)
	if err != nil {
		return nil, humautil.NewError(fmt.Errorf("GetTenants: failed to get tenants: %w", err), http.StatusInternalServerError)
	}

	tenants := make([]TenantResponse, len(dbTenants))
	for i, tenant := range dbTenants {

		enterpriseFeatures := authz.EnterpriseFeatures{}
		if len(tenant.EnterpriseFeatures) > 0 {
			if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
				return nil, humautil.NewError(fmt.Errorf("GetTenants: failed to unmarshal features config for tenant %s: %w", tenant.ID.String(), err), http.StatusInternalServerError)
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
