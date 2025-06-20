package ctx

import (
	"context"

	"dislyze/jirachi/authz"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const (
	TenantIDKey           contextKey = "tenant_id"
	UserIDKey             contextKey = "user_id"
	EnterpriseFeaturesKey contextKey = "enterprise_features"
	IsInternalUserKey     contextKey = "is_internal_user"
)

func GetTenantID(ctx context.Context) pgtype.UUID {
	tenantID := ctx.Value(TenantIDKey).(pgtype.UUID)
	return tenantID
}

func GetUserID(ctx context.Context) pgtype.UUID {
	userID := ctx.Value(UserIDKey).(pgtype.UUID)
	return userID
}

func WithEnterpriseFeatures(ctx context.Context, features *authz.EnterpriseFeatures) context.Context {
	return context.WithValue(ctx, EnterpriseFeaturesKey, features)
}

func GetEnterpriseFeatures(ctx context.Context) *authz.EnterpriseFeatures {
	features := ctx.Value(EnterpriseFeaturesKey).(*authz.EnterpriseFeatures)
	return features
}

func GetEnterpriseFeatureEnabled(ctx context.Context, featureName string) bool {
	features := GetEnterpriseFeatures(ctx)

	switch featureName {
	case "rbac":
		return features.RBAC.Enabled
	case "ip_whitelist":
		return features.IPWhitelist.Enabled
	default:
		return false
	}
}

func GetIPWhitelistConfig(ctx context.Context) *authz.IPWhitelist {
	features := GetEnterpriseFeatures(ctx)
	return &features.IPWhitelist
}

func WithIsInternalUser(ctx context.Context, isInternalUser bool) context.Context {
	return context.WithValue(ctx, IsInternalUserKey, isInternalUser)
}

func GetIsInternalUser(ctx context.Context) bool {
	isInternalUser := ctx.Value(IsInternalUserKey).(bool)
	return isInternalUser
}
