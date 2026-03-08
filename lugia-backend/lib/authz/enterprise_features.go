package authz

import (
	"context"

	libctx "dislyze/jirachi/ctx"
)

type EnterpriseFeature string

const (
	FeatureRBAC        EnterpriseFeature = "rbac"
	FeatureIPWhitelist EnterpriseFeature = "ip_whitelist"
	FeatureSSO         EnterpriseFeature = "sso"
	FeatureAuditLog    EnterpriseFeature = "audit_log"
)

func TenantHasFeature(ctx context.Context, feature EnterpriseFeature) bool {
	switch feature {
	case FeatureRBAC:
		return libctx.GetEnterpriseFeatureEnabled(ctx, "rbac")
	case FeatureIPWhitelist:
		return libctx.GetEnterpriseFeatureEnabled(ctx, "ip_whitelist")
	case FeatureSSO:
		return libctx.GetEnterpriseFeatureEnabled(ctx, "sso")
	case FeatureAuditLog:
		return libctx.GetEnterpriseFeatureEnabled(ctx, "audit_log")
	default:
		return false
	}
}
