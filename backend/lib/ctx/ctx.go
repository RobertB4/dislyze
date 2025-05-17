package ctx

import "context"

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

func GetTenantID(ctx context.Context) string {
	tenantID := ctx.Value(TenantIDKey).(string)
	return tenantID
}

func GetUserID(ctx context.Context) string {
	userID := ctx.Value(UserIDKey).(string)
	return userID
}
