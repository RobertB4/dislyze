package roles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"lugia/features/roles"
	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermissions_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map
		expectedStatus int
		expectUnauth   bool
		validateFunc   func(t *testing.T, response *roles.GetPermissionsResponse)
	}{
		{
			name:           "unauthenticated user gets 401",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "user without roles.view permission gets 403 forbidden",
			loginUserKey:   "enterprise_2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user with roles.view permission successfully retrieves all permissions",
			loginUserKey:   "enterprise_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetPermissionsResponse) {
				require.NotNil(t, response)
				assert.GreaterOrEqual(t, len(response.Permissions), 6, "Should have at least 6 permissions from seed data")

				// Validate each permission has required fields
				for _, permission := range response.Permissions {
					assert.NotEmpty(t, permission.ID, "Permission ID should not be empty")
					assert.NotEmpty(t, permission.Resource, "Permission resource should not be empty")
					assert.NotEmpty(t, permission.Action, "Permission action should not be empty")
					assert.NotEmpty(t, permission.Description, "Permission description should not be empty")
				}

				// Check for unique permissions (no duplicates)
				idSet := make(map[string]bool)
				for _, permission := range response.Permissions {
					assert.False(t, idSet[permission.ID], "Permission ID should be unique: %s", permission.ID)
					idSet[permission.ID] = true
				}
			},
		},
		{
			name:           "Internal admin user also gets all permissions (permissions are global)",
			loginUserKey:   "internal_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetPermissionsResponse) {
				require.NotNil(t, response)
				assert.GreaterOrEqual(t, len(response.Permissions), 6, "Should have at least 6 permissions for any tenant")
			},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// If success, validate response
			if tt.expectedStatus == http.StatusOK && tt.validateFunc != nil {
				var response roles.GetPermissionsResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err, "Should be able to decode JSON response")

				tt.validateFunc(t, &response)
			}
		})
	}
}

func TestGetPermissions_InvalidToken(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name        string
		tokenValue  string
		description string
	}{
		{
			name:        "malformed JWT token",
			tokenValue:  "invalid.jwt.token",
			description: "Should reject malformed JWT",
		},
		{
			name:        "empty token",
			tokenValue:  "",
			description: "Should reject empty token",
		},
		{
			name:        "random string token",
			tokenValue:  "random-string-not-jwt",
			description: "Should reject non-JWT token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: tt.tokenValue,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, tt.description)
		})
	}
}

func TestGetPermissions_ResponseFormat(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as enterprise admin
	userData := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, userData.Email, userData.PlainTextPassword)

	client := &http.Client{}

	// Make the request
	reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)
	req, err := http.NewRequest("GET", reqURL, nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "dislyze_access_token",
		Value: accessToken,
		Path:  "/",
	})

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response roles.GetPermissionsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Validate JSON structure
	require.NotNil(t, response.Permissions)
	assert.GreaterOrEqual(t, len(response.Permissions), 6, "Should have at least 6 permissions")

	// Validate each permission structure and content
	expectedResources := map[string]bool{"users": false, "roles": false, "tenant": false, "ip_whitelist": false, "audit_log": false}
	expectedActions := map[string]bool{"view": false, "edit": false}

	for _, permission := range response.Permissions {
		// Each permission should have required fields with correct types
		assert.NotEmpty(t, permission.ID, "Permission ID should not be empty")
		assert.NotEmpty(t, permission.Resource, "Permission resource should not be empty")
		assert.NotEmpty(t, permission.Action, "Permission action should not be empty")
		assert.NotEmpty(t, permission.Description, "Permission description should not be empty")

		// Validate resource is one of expected values
		assert.Contains(t, expectedResources, permission.Resource, "Resource should be users, roles, or tenant")
		expectedResources[permission.Resource] = true

		// Validate action is one of expected values
		assert.Contains(t, expectedActions, permission.Action, "Action should be view or edit")
		expectedActions[permission.Action] = true

		// Description should be Japanese
		assert.Greater(t, len(permission.Description), 0, "Permission description should not be empty")
	}

	// Verify we have all expected resources
	for resource, found := range expectedResources {
		assert.True(t, found, "Should have permissions for resource: %s", resource)
	}

	// Verify we have all expected actions
	for action, found := range expectedActions {
		assert.True(t, found, "Should have permissions with action: %s", action)
	}
}

func updateTenantEnterpriseFeatures(t *testing.T, pool *pgxpool.Pool, tenantID string, features map[string]interface{}) {
	t.Helper()
	featuresJSON, err := json.Marshal(features)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`UPDATE tenants SET enterprise_features = $1 WHERE id = $2`,
		featuresJSON, tenantID)
	require.NoError(t, err)
}

func TestGetPermissions_FeatureFiltering(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	smbTenant := setup.TestTenantsData["smb"]
	smbUser := setup.TestUsersData["smb_1"]

	t.Run("disabled features are excluded from permissions", func(t *testing.T) {
		setup.ResetAndSeedDB(t, pool)

		// Enable only RBAC for SMB tenant (no ip_whitelist, no audit_log)
		updateTenantEnterpriseFeatures(t, pool, smbTenant.ID, map[string]interface{}{
			"rbac":         map[string]interface{}{"enabled": true},
			"ip_whitelist": map[string]interface{}{"enabled": false},
		})

		accessToken, _ := setup.LoginUserAndGetTokens(t, smbUser.Email, smbUser.PlainTextPassword)

		reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)
		req, err := http.NewRequest("GET", reqURL, nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response roles.GetPermissionsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		resources := make(map[string]bool)
		for _, p := range response.Permissions {
			resources[p.Resource] = true
		}
		assert.False(t, resources["ip_whitelist"], "Should not include ip_whitelist permissions")
		assert.False(t, resources["audit_log"], "Should not include audit_log permissions")
		assert.True(t, resources["tenant"], "Should include core tenant permissions")
		assert.True(t, resources["users"], "Should include core users permissions")
		assert.True(t, resources["roles"], "Should include core roles permissions")
	})

	t.Run("enabled features are included in permissions", func(t *testing.T) {
		setup.ResetAndSeedDB(t, pool)

		enterpriseUser := setup.TestUsersData["enterprise_1"]
		accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseUser.Email, enterpriseUser.PlainTextPassword)

		reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)
		req, err := http.NewRequest("GET", reqURL, nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response roles.GetPermissionsResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		resources := make(map[string]bool)
		for _, p := range response.Permissions {
			resources[p.Resource] = true
		}
		assert.True(t, resources["ip_whitelist"], "Should include ip_whitelist permissions")
		assert.True(t, resources["audit_log"], "Should include audit_log permissions")
	})

	t.Run("enabling a feature makes its permissions visible", func(t *testing.T) {
		setup.ResetAndSeedDB(t, pool)

		// Start with only RBAC
		updateTenantEnterpriseFeatures(t, pool, smbTenant.ID, map[string]interface{}{
			"rbac":         map[string]interface{}{"enabled": true},
			"ip_whitelist": map[string]interface{}{"enabled": false},
		})

		accessToken, _ := setup.LoginUserAndGetTokens(t, smbUser.Email, smbUser.PlainTextPassword)

		reqURL := fmt.Sprintf("%s/roles/permissions", setup.BaseURL)

		// First request: feature-gated permissions excluded
		req1, err := http.NewRequest("GET", reqURL, nil)
		require.NoError(t, err)
		req1.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})
		resp1, err := client.Do(req1)
		require.NoError(t, err)

		var response1 roles.GetPermissionsResponse
		err = json.NewDecoder(resp1.Body).Decode(&response1)
		_ = resp1.Body.Close()
		require.NoError(t, err)

		resources1 := make(map[string]bool)
		for _, p := range response1.Permissions {
			resources1[p.Resource] = true
		}
		assert.False(t, resources1["audit_log"], "Should not include audit_log before enabling")
		assert.False(t, resources1["ip_whitelist"], "Should not include ip_whitelist")

		// Enable audit_log
		updateTenantEnterpriseFeatures(t, pool, smbTenant.ID, map[string]interface{}{
			"rbac":         map[string]interface{}{"enabled": true},
			"ip_whitelist": map[string]interface{}{"enabled": false},
			"audit_log":    map[string]interface{}{"enabled": true},
		})

		// Second request: audit_log now visible, ip_whitelist still excluded
		req2, err := http.NewRequest("GET", reqURL, nil)
		require.NoError(t, err)
		req2.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})
		resp2, err := client.Do(req2)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		var response2 roles.GetPermissionsResponse
		err = json.NewDecoder(resp2.Body).Decode(&response2)
		require.NoError(t, err)

		resources2 := make(map[string]bool)
		for _, p := range response2.Permissions {
			resources2[p.Resource] = true
		}
		assert.True(t, resources2["audit_log"], "Should now include audit_log permissions")
		assert.False(t, resources2["ip_whitelist"], "Should still not include ip_whitelist")
	})
}
