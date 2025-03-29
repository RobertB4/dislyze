package ctx

import "context"

type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

// GetTenantID retrieves the tenant_id from the context
func GetTenantID(ctx context.Context) string {
	tenantID := ctx.Value(TenantIDKey).(string)
	return tenantID
}

// GetUserID retrieves the user_id from the context
func GetUserID(ctx context.Context) string {
	userID := ctx.Value(UserIDKey).(string)
	return userID
}
