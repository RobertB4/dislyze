// Feature doc: docs/features/user-management.md
package tenants

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"giratina/lib/humautil"
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

var GetUsersByTenantOp = huma.Operation{
	OperationID: "get-users-by-tenant",
	Method:      http.MethodGet,
	Path:        "/tenants/{tenantID}/users",
}

type GetUsersByTenantInput struct {
	TenantID string `path:"tenantID"`
}

type GetUsersByTenantOutput struct {
	Body GetUsersByTenantResponse
}

func (h *TenantsHandler) GetUsersByTenant(ctx context.Context, input *GetUsersByTenantInput) (*GetUsersByTenantOutput, error) {
	var tenantID pgtype.UUID
	if err := tenantID.Scan(input.TenantID); err != nil {
		return nil, humautil.NewError(fmt.Errorf("invalid tenant ID format: %w", err), http.StatusBadRequest)
	}

	users, err := h.getUsersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &GetUsersByTenantOutput{Body: GetUsersByTenantResponse{Users: users}}, nil
}

func (h *TenantsHandler) getUsersByTenant(ctx context.Context, tenantID pgtype.UUID) ([]UserInfo, error) {
	dbUsers, err := h.queries.GetUsersByTenantID(ctx, tenantID)
	if err != nil {
		return nil, humautil.NewError(fmt.Errorf("GetUsersByTenant: failed to get users by tenant: %w", err), http.StatusInternalServerError)
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
