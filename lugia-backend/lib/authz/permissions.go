package authz

import (
	"context"

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
		errlib.LogError(errlib.New(err, 500, "failed to check user permission"))
		return false
	}
	return hasPermission
}
