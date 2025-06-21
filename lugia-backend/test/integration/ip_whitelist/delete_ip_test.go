package ip_whitelist

import (
	"fmt"
	"net/http"
	"testing"

	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestDeleteIPIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, pool *pgxpool.Pool) (string, string) // Returns (userKey, ipRuleID)
		expectedStatus int
	}{
		{
			name: "test_unauthenticated_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				return "", "11111111-1111-1111-1111-111111111111" // No user login, dummy ID
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_no_ip_whitelist_permission_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Use enterprise_2 who has editor role but no IP whitelist permissions
				return "enterprise_2", "11111111-1111-1111-1111-111111111111" // Dummy ID
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_feature_disabled_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Set enterprise features: ip_whitelist.enabled = false (feature disabled)
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     false, // Feature disabled - should block access
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_1", "11111111-1111-1111-1111-111111111111" // Dummy ID
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_invalid_uuid_returns_400",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Set enterprise features: enabled=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_1", "invalid-uuid"
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "test_ip_rule_does_not_exist_returns_404",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Set enterprise features: enabled=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_1", "99999999-9999-9999-9999-999999999999" // Non-existent ID
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "test_different_tenant_ip_returns_404",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create IP rule for enterprise tenant
				ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Enterprise IP", setup.TestUsersData["enterprise_1"].UserID)

				// Set enterprise features for SMB tenant
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["smb"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				// Return SMB user trying to delete enterprise IP
				return "smb_1", ipRuleID
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "test_delete_existing_ip_rule_success",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Set enterprise features: enabled=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				// Create IP rule for this tenant
				ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Test IP", setup.TestUsersData["enterprise_1"].UserID)

				return "enterprise_1", ipRuleID
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "test_delete_with_empty_request_body_success",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Set enterprise features: enabled=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				// Create IP rule for this tenant
				ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)

				return "enterprise_1", ipRuleID
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset database state before each test
			setup.ResetAndSeedDB(t, pool)

			userKey, ipRuleID := tt.setupFunc(t, pool)

			client := &http.Client{}

			// Create request with empty body
			reqURL := fmt.Sprintf("%s/ip-whitelist/%s/delete", setup.BaseURL, ipRuleID)
			req, err := http.NewRequest("POST", reqURL, nil)
			assert.NoError(t, err)

			// Add authentication if user is specified
			if userKey != "" {
				email, password := findUserCredentials(userKey)
				accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			// Make request
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
