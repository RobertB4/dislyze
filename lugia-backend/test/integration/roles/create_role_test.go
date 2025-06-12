package roles

import (
	"bytes"
	"encoding/json"
	"lugia/features/roles"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateRole_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	type createRoleTestCase struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map, empty for unauth
		requestBody    roles.CreateRoleRequestBody
		expectedStatus int
		expectUnauth   bool
	}

	tests := []createRoleTestCase{
		// Authentication & Authorization Tests
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Unauthorized Role",
				Description:   "This should fail",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "user without roles.create permission gets 403 forbidden",
			loginUserKey: "alpha_editor",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Forbidden Role",
				Description:   "This should be forbidden",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
			},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:         "validation error: missing name",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "",
				Description:   "Valid description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: no permissions",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid description",
				PermissionIDs: []string{}, // Empty permissions array
			},
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic Tests
		{
			name:         "error: duplicate role name in same tenant",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "管理者", // Same as existing admin role in tenant Alpha
				Description:   "Duplicate name role",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "error: invalid permission ID",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Role with Invalid Permission",
				Description:   "This has an invalid permission",
				PermissionIDs: []string{"99999999-9999-9999-9999-999999999999"}, // Non-existent permission
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "error: malformed permission UUID",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Role with Malformed UUID",
				Description:   "This has a malformed permission ID",
				PermissionIDs: []string{"not-a-valid-uuid"},
			},
			expectedStatus: http.StatusInternalServerError,
		},

		// Success Tests
		{
			name:         "success: create role with single permission",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Test Role Single",
				Description:   "A test role with one permission",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: create role with multiple permissions",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:        "Test Role Multi",
				Description: "A test role with multiple permissions",
				PermissionIDs: []string{
					"3a52c807-ddcb-4044-8682-658e04800a8e", // users.view
					"db994eda-6ff7-4ae5-a675-3abe735ce9cc", // users.edit
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: create role with empty description",
			loginUserKey: "alpha_admin",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "No Description Role",
				Description:   "",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			jsonData, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", setup.BaseURL+"/roles/create", bytes.NewBuffer(jsonData))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Add authentication if needed
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s for test: %s", tt.loginUserKey, tt.name)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
				})
			}

			// Execute request
			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			// Verify response status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Test: %s", tt.name)
		})
	}
}
