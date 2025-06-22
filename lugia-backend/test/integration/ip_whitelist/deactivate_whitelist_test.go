package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateWhitelistIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	client := &http.Client{}

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, pool *pgxpool.Pool) (userKey string, shouldAddToWhitelist bool)
		expectedStatus int
		validateFunc   func(t *testing.T, pool *pgxpool.Pool)
	}{
		{
			name: "test_unauthenticated_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				return "", false // No user login
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_no_ip_whitelist_permission_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_2", true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_feature_disabled_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				// Set enterprise features: ip_whitelist.enabled = false
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     false, // Feature disabled
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_ip_whitelist_view_permission_insufficient",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				// Use enterprise_2 who has editor role but no IP whitelist permissions (similar to view-only scenario)
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_2", true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_deactivate_when_active_succeeds",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				// enterprise_1 already has IP whitelist permissions from seed data
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true, // Currently active
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", true
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "test_deactivate_when_inactive_idempotent",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false, // Already inactive
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", true
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "test_tenant_isolation",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, bool) {
				// Set up both enterprise and SMB tenants with active whitelists
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["smb"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": false,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true, // SMB also has active whitelist
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", true
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool) {
				// Verify enterprise tenant is deactivated
				var enterpriseFeatures map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err := row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &enterpriseFeatures)
				require.NoError(t, err)
				assert.False(t, enterpriseFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))

				// Verify SMB tenant remains active
				var smbFeatures map[string]interface{}
				row = pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["smb"].ID)
				err = row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &smbFeatures)
				require.NoError(t, err)
				assert.True(t, smbFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup.ResetAndSeedDB(t, pool)

			userKey, shouldAddToWhitelist := tt.setupFunc(t, pool)

			// Add user IP to whitelist if needed (so middleware allows request)
			if shouldAddToWhitelist && userKey != "" {
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.100", "Test IP", setup.TestUsersData[userKey].UserID)
			}

			// Create request
			reqURL := fmt.Sprintf("%s/ip-whitelist/deactivate", setup.BaseURL)
			req, err := http.NewRequest("POST", reqURL, nil)
			require.NoError(t, err)

			// Set client IP header to simulate user request from specific IP
			req.Header.Set("X-Real-IP", "192.168.1.100")

			// Add auth if user specified
			if userKey != "" {
				email, password := findUserCredentials(userKey)
				accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Verify status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Run additional validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, pool)
			}
		})
	}
}
