// Feature doc: docs/features/rbac.md, docs/features/ip-whitelisting.md
package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"giratina/queries"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/authz"
	"giratina/lib/humautil"
)

var UpdateTenantOp = huma.Operation{
	OperationID: "update-tenant",
	Method:      http.MethodPost,
	Path:        "/tenants/{id}/update",
}

type UpdateTenantInput struct {
	ID   string `path:"id"`
	Body UpdateTenantRequestBody
}

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

func (h *TenantsHandler) UpdateTenant(ctx context.Context, input *UpdateTenantInput) (*struct{}, error) {
	var tenantID pgtype.UUID
	if err := tenantID.Scan(input.ID); err != nil {
		return nil, humautil.NewError(fmt.Errorf("invalid tenant ID format: %w", err), http.StatusBadRequest)
	}

	if err := input.Body.Validate(); err != nil {
		return nil, humautil.NewError(fmt.Errorf("update tenant validation failed: %w", err), http.StatusBadRequest)
	}

	if err := h.updateTenant(ctx, &tenantID, &input.Body); err != nil {
		return nil, err
	}

	return nil, nil
}

func (h *TenantsHandler) updateTenant(ctx context.Context, tenantID *pgtype.UUID, requestBody *UpdateTenantRequestBody) error {
	enterpriseFeaturesJSON, err := json.Marshal(requestBody.EnterpriseFeatures)
	if err != nil {
		return humautil.NewError(fmt.Errorf("failed to marshal enterprise features: %w", err), http.StatusInternalServerError)
	}

	err = h.queries.UpdateTenant(ctx, &queries.UpdateTenantParams{
		Name:               requestBody.Name,
		EnterpriseFeatures: enterpriseFeaturesJSON,
		ID:                 *tenantID,
	})
	if err != nil {
		return humautil.NewError(fmt.Errorf("failed to update tenant: %w", err), http.StatusInternalServerError)
	}

	return nil
}
