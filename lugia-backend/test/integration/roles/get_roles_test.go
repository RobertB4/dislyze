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
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map
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
			name:           "user without roles.create permission gets 403 forbidden",
			loginUserKey:   "alpha_editor",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user with roles.create permission successfully retrieves tenant roles (Tenant Alpha)",
			loginUserKey:   "alpha_admin",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetRolesResponse) {
				require.NotNil(t, response)
				require.Len(t, response.Roles, 2, "Should have exactly 2 roles for Tenant Alpha")

				// Roles should be ordered by name: "管理者" comes before "編集者" in PostgreSQL ordering
				adminRole := response.Roles[0]
				editorRole := response.Roles[1]

				// Validate admin role (all permissions) - Tenant Alpha admin role
				assert.Equal(t, "e0000000-0000-0000-0000-000000000001", adminRole.ID, "Should get Tenant Alpha admin role ID")
				assert.Equal(t, "管理者", adminRole.Name)
				assert.Equal(t, "すべての管理機能にアクセス可能", adminRole.Description)
				assert.Len(t, adminRole.Permissions, 9, "Admin role should have 9 permissions")

				// Validate editor role (no permissions) - Tenant Alpha editor role
				assert.Equal(t, "e0000000-0000-0000-0000-000000000002", editorRole.ID, "Should get Tenant Alpha editor role ID")
				assert.Equal(t, "編集者", editorRole.Name)
				assert.Equal(t, "限定的な編集権限", editorRole.Description)
				assert.Len(t, editorRole.Permissions, 0, "Editor role should have no permissions")

				// Permissions should be ordered by description alphabetically
				expectedPermissions := []string{
					"テナント設定の変更",
					"ユーザーの削除",
					"ユーザーの招待",
					"ユーザー一覧の閲覧",
					"ユーザー権限の変更",
					"ロールの作成",
					"ロールの削除",
					"ロールの編集",
					"ロール一覧の閲覧",
				}
				assert.Equal(t, expectedPermissions, adminRole.Permissions)
			},
		},
		{
			name:           "beta admin user successfully retrieves tenant roles (Tenant Beta)",
			loginUserKey:   "beta_admin",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *roles.GetRolesResponse) {
				require.NotNil(t, response)
				require.Len(t, response.Roles, 2, "Should have exactly 2 roles for Tenant Beta")

				// Both roles should exist but only beta admin role should have permissions
				adminRole := response.Roles[0]
				editorRole := response.Roles[1]

				// Validate Tenant Beta admin role - should have different ID than Alpha
				assert.Equal(t, "e0000000-0000-0000-0000-000000000003", adminRole.ID, "Should get Tenant Beta admin role ID")
				assert.Equal(t, "管理者", adminRole.Name)
				assert.Len(t, adminRole.Permissions, 9, "Beta admin role should have 9 permissions")

				// Validate Tenant Beta editor role - should have different ID than Alpha
				assert.Equal(t, "e0000000-0000-0000-0000-000000000004", editorRole.ID, "Should get Tenant Beta editor role ID")
				assert.Equal(t, "編集者", editorRole.Name)
				assert.Len(t, editorRole.Permissions, 0, "Beta editor role should have no permissions")
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
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as alpha admin
	userData := setup.TestUsersData["alpha_admin"]
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
	require.Len(t, response.Roles, 2)

	for _, role := range response.Roles {
		// Each role should have required fields
		assert.NotEmpty(t, role.ID, "Role ID should not be empty")
		assert.NotEmpty(t, role.Name, "Role name should not be empty")
		assert.NotEmpty(t, role.Description, "Role description should not be empty")
		assert.NotNil(t, role.Permissions, "Permissions array should not be nil")

		// Validate permission format
		for _, permission := range role.Permissions {
			assert.NotEmpty(t, permission, "Permission should not be empty")
			// Permissions should be Japanese descriptions
			assert.Greater(t, len(permission), 0, "Permission description should not be empty")
		}
	}
}
