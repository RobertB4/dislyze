// Feature doc: docs/features/profile-management.md
package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

var GetMeOp = huma.Operation{
	OperationID: "get-me",
	Method:      http.MethodGet,
	Path:        "/me",
}

// ClientEnterpriseFeatures contains only the enterprise features safe to expose to clients
type ClientEnterpriseFeatures struct {
	RBAC        authz.RBAC        `json:"rbac"`
	IPWhitelist authz.IPWhitelist `json:"ip_whitelist"`
	AuditLog    authz.AuditLog    `json:"audit_log"`
}

type MeResponse struct {
	TenantName         string                   `json:"tenant_name"`
	UserID             string                   `json:"user_id"`
	Email              string                   `json:"email"`
	UserName           string                   `json:"user_name"`
	Permissions        []string                 `json:"permissions" nullable:"false"`
	EnterpriseFeatures ClientEnterpriseFeatures `json:"enterprise_features"`
}

type GetMeInput struct{}

type GetMeOutput struct {
	Body MeResponse
}

func (h *UsersHandler) GetMe(ctx context.Context, input *GetMeInput) (*GetMeOutput, error) {
	response, err := h.getMe(ctx)
	if err != nil {
		return nil, err
	}
	return &GetMeOutput{Body: *response}, nil
}

func (h *UsersHandler) getMe(ctx context.Context) (*MeResponse, error) {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	user, err := h.q.GetUserByID(ctx, userID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.NewError(fmt.Errorf("GetMe: user not found %s: %w", userID.String(), err), http.StatusUnauthorized)
		}
		return nil, errlib.NewError(fmt.Errorf("GetMe: failed to get user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	tenant, err := h.q.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return nil, errlib.NewError(fmt.Errorf("GetMe: tenant not found %s for user %s: %w", tenantID.String(), userID.String(), err), http.StatusUnauthorized)
		}
		return nil, errlib.NewError(fmt.Errorf("GetMe: failed to get tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError)
	}

	var enterpriseFeatures authz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		return nil, errlib.NewError(fmt.Errorf("GetMe: failed to unmarshal features config for tenant %s: %w", tenantID.String(), err), http.StatusInternalServerError)
	}

	permissionRows, err := h.q.GetUserPermissionsWithFallback(ctx, &queries.GetUserPermissionsWithFallbackParams{
		UserID:      userID,
		TenantID:    tenantID,
		RbacEnabled: enterpriseFeatures.RBAC.Enabled,
	})
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("GetMe: failed to get user permissions for user %s: %w", userID.String(), err), http.StatusInternalServerError)
	}

	permissionsRes := make([]string, len(permissionRows))
	for i, row := range permissionRows {
		permissionsRes[i] = fmt.Sprintf("%s.%s", row.Resource, row.Action)
	}

	response := &MeResponse{
		TenantName:  tenant.Name,
		UserID:      user.ID.String(),
		Email:       user.Email,
		UserName:    user.Name,
		Permissions: permissionsRes,
		EnterpriseFeatures: ClientEnterpriseFeatures{
			RBAC:        enterpriseFeatures.RBAC,
			IPWhitelist: enterpriseFeatures.IPWhitelist,
			AuditLog:    enterpriseFeatures.AuditLog,
		},
	}

	return response, nil
}
