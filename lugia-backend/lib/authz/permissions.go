package authz

import (
	"context"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

const (
	ResourceTenant     = "tenant"
	ResourceUsers      = "users"
	ResourceRoles      = "roles"
	ResourceIPWhitelist = "ip_whitelist"
)

const (
	ActionView = "view"
	ActionEdit = "edit"
)

func UserHasPermission(ctx context.Context, db *queries.Queries, resource, action string) bool {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	rbacEnabled := libctx.GetEnterpriseFeatureEnabled(ctx, "rbac")

	hasPermission, err := db.UserHasPermission(ctx, &queries.UserHasPermissionParams{
		UserID:      userID,
		TenantID:    tenantID,
		Resource:    resource,
		Action:      action,
		RbacEnabled: rbacEnabled,
	})
	if err != nil {
		errlib.LogError(errlib.New(err, 500, "failed to check user permission"))
		return false
	}
	return hasPermission
}
