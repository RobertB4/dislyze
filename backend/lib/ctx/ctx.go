package ctx

import (
	"context"
	"dislyze/lib/middleware"

	"github.com/jackc/pgx/v5/pgtype"
)

func GetTenantID(ctx context.Context) pgtype.UUID {
	tenantID := ctx.Value(middleware.TenantIDKey).(pgtype.UUID)
	return tenantID
}

func GetUserID(ctx context.Context) pgtype.UUID {
	userID := ctx.Value(middleware.UserIDKey).(pgtype.UUID)
	return userID
}
