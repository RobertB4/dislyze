package authz

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/queries"
)

const (
	FeatureRBAC = "rbac"
)

type EnterpriseFeatures struct {
	RBAC RBAC `json:"rbac"`
}

type RBAC struct {
	Enabled bool `json:"enabled"`
}

func TenantHasFeature(ctx context.Context, db *queries.Queries, feature string) bool {
	tenantID := libctx.GetTenantID(ctx)

	tenant, err := db.GetTenantByID(ctx, tenantID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			errlib.LogError(errlib.New(err, 500, "tenant not found when checking feature"))
		} else {
			errlib.LogError(errlib.New(err, 500, "failed to get tenant when checking feature"))
		}
		return false
	}

	var enterpriseFeatures EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &enterpriseFeatures); err != nil {
		errlib.LogError(errlib.New(err, 500, "failed to unmarshal features config"))
		return false
	}

	switch feature {
	case FeatureRBAC:
		return enterpriseFeatures.RBAC.Enabled
	default:
		return false
	}
}
