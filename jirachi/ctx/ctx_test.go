package ctx

import (
	"context"
	"testing"

	"dislyze/jirachi/authz"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestGetTenantID(t *testing.T) {
	t.Run("returns tenant ID from context", func(t *testing.T) {
		tenantID := pgtype.UUID{
			Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			Valid: true,
		}
		ctx := context.WithValue(context.Background(), TenantIDKey, tenantID)

		got := GetTenantID(ctx)

		assert.Equal(t, tenantID, got)
	})

	t.Run("panics when tenant ID is missing from context", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			GetTenantID(ctx)
		})
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("returns user ID from context", func(t *testing.T) {
		userID := pgtype.UUID{
			Bytes: [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			Valid: true,
		}
		ctx := context.WithValue(context.Background(), UserIDKey, userID)

		got := GetUserID(ctx)

		assert.Equal(t, userID, got)
	})

	t.Run("panics when user ID is missing from context", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			GetUserID(ctx)
		})
	})
}

func TestEnterpriseFeatures(t *testing.T) {
	t.Run("round-trip set and get", func(t *testing.T) {
		features := &authz.EnterpriseFeatures{
			RBAC:        authz.RBAC{Enabled: true},
			IPWhitelist: authz.IPWhitelist{Enabled: false, Active: true},
			SSO:         authz.SSO{Enabled: true, IdpMetadataURL: "https://idp.example.com"},
		}
		ctx := WithEnterpriseFeatures(context.Background(), features)

		got := GetEnterpriseFeatures(ctx)

		assert.Equal(t, features, got)
	})

	t.Run("panics when enterprise features missing from context", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			GetEnterpriseFeatures(ctx)
		})
	})
}

func TestGetEnterpriseFeatureEnabled(t *testing.T) {
	features := &authz.EnterpriseFeatures{
		RBAC:        authz.RBAC{Enabled: true},
		IPWhitelist: authz.IPWhitelist{Enabled: true},
		SSO:         authz.SSO{Enabled: false},
	}

	tests := []struct {
		name        string
		featureName string
		want        bool
	}{
		{
			name:        "rbac enabled",
			featureName: "rbac",
			want:        true,
		},
		{
			name:        "ip_whitelist enabled",
			featureName: "ip_whitelist",
			want:        true,
		},
		{
			name:        "sso disabled",
			featureName: "sso",
			want:        false,
		},
		{
			name:        "unknown feature returns false",
			featureName: "nonexistent",
			want:        false,
		},
		{
			name:        "empty string returns false",
			featureName: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithEnterpriseFeatures(context.Background(), features)

			got := GetEnterpriseFeatureEnabled(ctx, tt.featureName)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetIPWhitelistConfig(t *testing.T) {
	t.Run("returns ip whitelist config from context", func(t *testing.T) {
		features := &authz.EnterpriseFeatures{
			IPWhitelist: authz.IPWhitelist{
				Enabled:                  true,
				Active:                   true,
				AllowInternalAdminBypass: true,
			},
		}
		ctx := WithEnterpriseFeatures(context.Background(), features)

		got := GetIPWhitelistConfig(ctx)

		assert.True(t, got.Enabled)
		assert.True(t, got.Active)
		assert.True(t, got.AllowInternalAdminBypass)
	})

	t.Run("returns pointer to original struct field", func(t *testing.T) {
		features := &authz.EnterpriseFeatures{
			IPWhitelist: authz.IPWhitelist{Enabled: false},
		}
		ctx := WithEnterpriseFeatures(context.Background(), features)

		got := GetIPWhitelistConfig(ctx)

		assert.False(t, got.Enabled)
	})
}

func TestIsInternalUser(t *testing.T) {
	t.Run("round-trip true", func(t *testing.T) {
		ctx := WithIsInternalUser(context.Background(), true)

		assert.True(t, GetIsInternalUser(ctx))
	})

	t.Run("round-trip false", func(t *testing.T) {
		ctx := WithIsInternalUser(context.Background(), false)

		assert.False(t, GetIsInternalUser(ctx))
	})

	t.Run("panics when not set", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			GetIsInternalUser(ctx)
		})
	})
}
