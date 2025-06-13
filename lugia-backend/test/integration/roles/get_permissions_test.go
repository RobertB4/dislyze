package roles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"lugia/features/roles"
	"lugia/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermissions_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData2 map
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
				loginDetails, ok := setup.TestUsersData2[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData2: %s", tt.loginUserKey)

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
	setup.ResetAndSeedDB2(t, pool)
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
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Login as enterprise admin
	userData := setup.TestUsersData2["enterprise_1"]
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
	expectedResources := map[string]bool{"users": false, "roles": false, "tenant": false}
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