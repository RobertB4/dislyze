package authz

import (
	"context"
	"fmt"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

type Resource string

func (r *Resource) String() string {
	if r == nil {
		return ""
	}
	return string(*r)
}

const (
	ResourceTenant      Resource = "tenant"
	ResourceUsers       Resource = "users"
	ResourceRoles       Resource = "roles"
	ResourceIPWhitelist Resource = "ip_whitelist"
	ResourceAuditLog    Resource = "audit_log"
)

const (
	ActionView = "view"
	ActionEdit = "edit"
)

func UserHasPermission(ctx context.Context, db *queries.Queries, resource Resource, action string) bool {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	rbacEnabled := libctx.GetEnterpriseFeatureEnabled(ctx, "rbac")

	hasPermission, err := db.UserHasPermission(ctx, &queries.UserHasPermissionParams{
		UserID:      userID,
		TenantID:    tenantID,
		Resource:    resource.String(),
		Action:      action,
		RbacEnabled: rbacEnabled,
	})
	if err != nil {
		errlib.LogError(fmt.Errorf("failed to check user permission: %w", err))
		return false
	}
	return hasPermission
}
