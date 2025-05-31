package ctx

import (
	"context"
	"lugia/queries_pregeneration"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

func GetTenantID(ctx context.Context) pgtype.UUID {
	tenantID := ctx.Value(TenantIDKey).(pgtype.UUID)
	return tenantID
}

func GetUserID(ctx context.Context) pgtype.UUID {
	userID := ctx.Value(UserIDKey).(pgtype.UUID)
	return userID
}

func GetUserRole(ctx context.Context) queries_pregeneration.UserRole {
	userRole, ok := ctx.Value(UserRoleKey).(queries_pregeneration.UserRole)
	if !ok {
		return queries_pregeneration.UserRole("")
	}
	return userRole
}
