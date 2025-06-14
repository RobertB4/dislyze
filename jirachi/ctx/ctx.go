package ctx

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

func GetTenantID(ctx context.Context) pgtype.UUID {
	tenantID := ctx.Value(TenantIDKey).(pgtype.UUID)
	return tenantID
}

func GetUserID(ctx context.Context) pgtype.UUID {
	userID := ctx.Value(UserIDKey).(pgtype.UUID)
	return userID
}