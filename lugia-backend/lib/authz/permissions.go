package authz

import (
	"context"

	libctx "lugia/lib/ctx"
	"lugia/lib/errlib"
	"lugia/queries"
)

const (
	ResourceTenant = "tenant"
	ResourceUsers  = "users"
	ResourceRoles  = "roles"
)

const (
	ActionView = "view"
	ActionEdit = "edit"
)

func UserHasPermission(ctx context.Context, db *queries.Queries, resource, action string) bool {
	userID := libctx.GetUserID(ctx)
	tenantID := libctx.GetTenantID(ctx)

	hasPermission, err := db.UserHasPermission(ctx, &queries.UserHasPermissionParams{
		UserID:   userID,
		TenantID: tenantID,
		Resource: resource,
		Action:   action,
	})
	if err != nil {
		errlib.LogError(errlib.New(err, 500, "failed to check user permission"))
		return false
	}
	return hasPermission
}
