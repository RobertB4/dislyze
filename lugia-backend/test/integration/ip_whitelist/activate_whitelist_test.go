package ip_whitelist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"lugia/features/ip_whitelist"
	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivateWhitelistIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	client := &http.Client{}

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, pool *pgxpool.Pool) (userKey string, requestBody map[string]interface{}, addUserIP bool)
		expectedStatus int
		validateFunc   func(t *testing.T, pool *pgxpool.Pool, responseBody []byte)
	}{
		{
			name: "test_unauthenticated_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
				return "", map[string]interface{}{}, false // No user login
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_no_ip_whitelist_permission_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
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
				return "enterprise_2", map[string]interface{}{}, false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_feature_disabled_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     false, // Feature disabled
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", map[string]interface{}{}, false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_invalid_json_returns_400",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
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
				// Return nil to signal we'll send malformed JSON manually
				return "enterprise_1", nil, false
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "test_unsafe_activation_returns_user_ip",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
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
				// Don't add user IP to whitelist - this makes activation unsafe
				return "enterprise_1", map[string]interface{}{"force": false}, false
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool, responseBody []byte) {
				// Should return user IP since activation is unsafe
				var response ip_whitelist.ActivateWhitelistResponse
				err := json.Unmarshal(responseBody, &response)
				require.NoError(t, err)
				assert.Equal(t, "192.168.1.100", response.UserIP)

				// Verify whitelist was NOT activated
				var features map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err = row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &features)
				require.NoError(t, err)
				assert.False(t, features["ip_whitelist"].(map[string]interface{})["active"].(bool))
			},
		},
		{
			name: "test_safe_activation_succeeds",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
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
				// Add user IP to whitelist - this makes activation safe
				return "enterprise_1", map[string]interface{}{"force": false}, true
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool, responseBody []byte) {
				// Should have no response body for successful activation
				assert.Empty(t, responseBody)

				// Verify whitelist was activated
				var features map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err := row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &features)
				require.NoError(t, err)
				assert.True(t, features["ip_whitelist"].(map[string]interface{})["active"].(bool))
			},
		},
		{
			name: "test_force_activation_bypasses_safety",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
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
				// Don't add user IP (would be unsafe), but force=true should bypass safety
				return "enterprise_1", map[string]interface{}{"force": true}, false
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool, responseBody []byte) {
				// Should have no response body for successful activation
				assert.Empty(t, responseBody)

				// Verify whitelist was activated despite being unsafe
				var features map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err := row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &features)
				require.NoError(t, err)
				assert.True(t, features["ip_whitelist"].(map[string]interface{})["active"].(bool))

				// Verify emergency token was created
				var tokenCount int
				err = pool.QueryRow(context.Background(),
					"SELECT COUNT(*) FROM ip_whitelist_emergency_tokens").Scan(&tokenCount)
				require.NoError(t, err)
				assert.Greater(t, tokenCount, 0, "Emergency token should have been created")
			},
		},
		{
			name: "test_tenant_isolation",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, map[string]interface{}, bool) {
				// Set up both enterprise and SMB tenants
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
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["smb"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": false,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false, // SMB starts inactive
						"allow_internal_admin_bypass": false,
					},
				})
				return "enterprise_1", map[string]interface{}{"force": false}, true
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool, responseBody []byte) {
				// Verify enterprise tenant is activated
				var enterpriseFeatures map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err := row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &enterpriseFeatures)
				require.NoError(t, err)
				assert.True(t, enterpriseFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))

				// Verify SMB tenant remains inactive
				var smbFeatures map[string]interface{}
				row = pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["smb"].ID)
				err = row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &smbFeatures)
				require.NoError(t, err)
				assert.False(t, smbFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup.ResetAndSeedDB(t, pool)

			userKey, requestBody, addUserIP := tt.setupFunc(t, pool)

			// Add user IP to whitelist if needed (so activation is "safe")
			if addUserIP && userKey != "" {
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.100", "Test IP", setup.TestUsersData[userKey].UserID)
			}

			// Create request
			reqURL := fmt.Sprintf("%s/ip-whitelist/activate", setup.BaseURL)
			var req *http.Request
			var err error

			// Handle special case for invalid JSON test
			if tt.name == "test_invalid_json_returns_400" {
				malformedJSON := `{"force": true` // Missing closing brace
				req, err = http.NewRequest("POST", reqURL, bytes.NewBuffer([]byte(malformedJSON)))
			} else {
				bodyBytes, marshalErr := json.Marshal(requestBody)
				require.NoError(t, marshalErr)
				req, err = http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
			}
			require.NoError(t, err)

			// Set headers
			req.Header.Set("Content-Type", "application/json")
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

			// Read response body
			var responseBody []byte
			if resp.ContentLength > 0 {
				responseBody = make([]byte, resp.ContentLength)
				_, err = resp.Body.Read(responseBody)
				if err != nil && err.Error() != "EOF" {
					require.NoError(t, err)
				}
			}

			// Verify status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Run additional validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, pool, responseBody)
			}
		})
	}
}