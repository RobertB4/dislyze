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

func TestGetRoles_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData2 map
		expectedStatus int
		expectUnauth   bool
		validateFunc   func(t *testing.T, response *roles.GetRolesResponse)
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
			name:           "user with roles.view permission successfully retrieves tenant roles (Enterprise tenant)",
			loginUserKey:   "enterprise_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetRolesResponse) {
				require.NotNil(t, response)
				assert.GreaterOrEqual(t, len(response.Roles), 3, "Enterprise tenant should have at least 3 default roles")

				// Find admin role by name
				var adminRole *roles.RoleInfo
				for i := range response.Roles {
					if response.Roles[i].Name == "管理者" {
						adminRole = &response.Roles[i]
						break
					}
				}
				require.NotNil(t, adminRole, "Should find admin role")

				// Validate admin role (should have permissions)
				assert.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", adminRole.ID, "Should get Enterprise admin role ID")
				assert.Equal(t, "管理者", adminRole.Name)
				assert.Equal(t, "すべての機能にアクセス可能", adminRole.Description)
				assert.True(t, adminRole.IsDefault, "Admin role should be default")
				assert.GreaterOrEqual(t, len(adminRole.Permissions), 3, "Admin role should have at least 3 permissions")

				// Check that each permission has the required fields
				for _, perm := range adminRole.Permissions {
					assert.NotEmpty(t, perm.ID, "Permission ID should not be empty")
					assert.NotEmpty(t, perm.Resource, "Permission resource should not be empty")
					assert.NotEmpty(t, perm.Action, "Permission action should not be empty")
					assert.NotEmpty(t, perm.Description, "Permission description should not be empty")
				}

				// Verify we have expected role names
				roleNames := make(map[string]bool)
				for _, role := range response.Roles {
					roleNames[role.Name] = true
				}
				assert.True(t, roleNames["管理者"], "Should have admin role")
				assert.True(t, roleNames["編集者"], "Should have editor role")
				assert.True(t, roleNames["閲覧者"], "Should have viewer role")
			},
		},
		{
			name:           "internal admin user successfully retrieves tenant roles (Internal tenant)",
			loginUserKey:   "internal_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetRolesResponse) {
				require.NotNil(t, response)
				assert.GreaterOrEqual(t, len(response.Roles), 3, "Internal tenant should have at least 3 default roles")

				// Find admin role by name
				var adminRole *roles.RoleInfo
				for i := range response.Roles {
					if response.Roles[i].Name == "管理者" {
						adminRole = &response.Roles[i]
						break
					}
				}
				require.NotNil(t, adminRole, "Should find admin role")

				// Validate Internal tenant admin role - should have different ID than Enterprise
				assert.Equal(t, "22222222-3333-4444-5555-666666666666", adminRole.ID, "Should get Internal admin role ID")
				assert.Equal(t, "管理者", adminRole.Name)
				assert.GreaterOrEqual(t, len(adminRole.Permissions), 3, "Internal admin role should have at least 3 permissions")

				// Verify tenant isolation - should not see Enterprise tenant roles
				for _, role := range response.Roles {
					assert.NotEqual(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", role.ID, "Should not see Enterprise admin role")
					assert.NotEqual(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", role.ID, "Should not see Enterprise editor role")
				}
			},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/roles", setup.BaseURL)
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
				var response roles.GetRolesResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err, "Should be able to decode JSON response")

				tt.validateFunc(t, &response)
			}
		})
	}
}

func TestGetRoles_ResponseFormat(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Login as enterprise admin
	userData := setup.TestUsersData2["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, userData.Email, userData.PlainTextPassword)

	client := &http.Client{}

	// Make the request
	reqURL := fmt.Sprintf("%s/roles", setup.BaseURL)
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

	var response roles.GetRolesResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Validate JSON structure
	require.NotNil(t, response.Roles)
	assert.GreaterOrEqual(t, len(response.Roles), 3, "Enterprise tenant should have at least 3 roles")

	for _, role := range response.Roles {
		// Each role should have required fields
		assert.NotEmpty(t, role.ID, "Role ID should not be empty")
		assert.NotEmpty(t, role.Name, "Role name should not be empty")
		assert.NotEmpty(t, role.Description, "Role description should not be empty")
		assert.NotNil(t, role.Permissions, "Permissions array should not be nil")

		// Validate permission format
		for _, permission := range role.Permissions {
			assert.NotEmpty(t, permission.ID, "Permission ID should not be empty")
			assert.NotEmpty(t, permission.Resource, "Permission resource should not be empty")
			assert.NotEmpty(t, permission.Action, "Permission action should not be empty")
			assert.NotEmpty(t, permission.Description, "Permission description should not be empty")
			// Permissions should be Japanese descriptions
			assert.Greater(t, len(permission.Description), 0, "Permission description should not be empty")
		}
	}
}
