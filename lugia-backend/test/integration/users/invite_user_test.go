package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var expectedInviteErrorMessages = map[string]string{
	"emailConflict": "このメールアドレスは既に使用されています。",
	"invalidRoles":  "一部のロールが無効です。",
}

func TestInviteUser_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	type inviteUserTestCase struct {
		name                   string
		loginUserKey           string // Key for setup.TestUsersData map, empty for unauth
		requestBody            users.InviteUserRequestBody
		expectedStatus         int
		expectedErrorKey       string // Key for expectedInviteErrorMessages, if any
		expectUnauth           bool
		validateRoleAssignment bool // Check that roles were assigned correctly
		expectedRoleCount      int
	}

	tests := []inviteUserTestCase{
		// Authentication & Authorization Tests
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			requestBody: users.InviteUserRequestBody{
				Email:   "unauth_invitee@example.com",
				Name:    "Unauth Invitee",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"}, // Alpha editor role
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "user without users.create permission gets 403 forbidden",
			loginUserKey: "alpha_editor", // Only has editor role, no users.create permission
			requestBody: users.InviteUserRequestBody{
				Email:   "forbidden_invitee@example.com",
				Name:    "Forbidden Invitee",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"}, // Alpha editor role
			},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:         "validation error: missing email",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "",
				Name:    "Test Name",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: invalid email format",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "invalid-email",
				Name:    "Test Name",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: missing name",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "valid@example.com",
				Name:    "",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: name with only whitespace",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "whitespace@example.com",
				Name:    "   ",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: missing role_ids (empty array)",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "valid@example.com",
				Name:    "Test Name",
				RoleIDs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: missing role_ids (nil)",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "valid@example.com",
				Name:    "Test Name",
				RoleIDs: nil,
			},
			expectedStatus: http.StatusBadRequest,
		},

		// Role ID Validation Tests
		{
			name:         "validation error: invalid UUID format in role_ids",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "valid@example.com",
				Name:    "Test Name",
				RoleIDs: []string{"invalid-uuid-format"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: non-existent role ID",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "valid@example.com",
				Name:    "Test Name",
				RoleIDs: []string{"f0000000-0000-0000-0000-000000000999"}, // Non-existent role
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorKey: "invalidRoles",
		},
		{
			name:         "security: role ID from different tenant",
			loginUserKey: "alpha_admin", // Tenant Alpha admin
			requestBody: users.InviteUserRequestBody{
				Email:   "cross_tenant@example.com",
				Name:    "Cross Tenant Test",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000003"}, // Tenant Beta admin role
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorKey: "invalidRoles",
		},
		{
			name:         "security: mix of valid and invalid role IDs (cross-tenant)",
			loginUserKey: "alpha_admin", // Tenant Alpha admin
			requestBody: users.InviteUserRequestBody{
				Email: "mixed_roles@example.com",
				Name:  "Mixed Roles Test",
				RoleIDs: []string{
					"e0000000-0000-0000-0000-000000000001", // Valid: Alpha admin role
					"e0000000-0000-0000-0000-000000000003", // Invalid: Beta admin role
				},
			},
			expectedStatus:   http.StatusBadRequest,
			expectedErrorKey: "invalidRoles",
		},

		// Business Logic Tests
		{
			name:         "error when email already exists",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   setup.TestUsersData["alpha_editor"].Email,
				Name:    "Duplicate Invitee",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"}, // Alpha editor role
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorKey: "emailConflict",
		},
		{
			name:         "successful invitation with single role",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "single_role@example.com",
				Name:    "Single Role User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000002"}, // Alpha editor role
			},
			expectedStatus:         http.StatusOK,
			validateRoleAssignment: true,
			expectedRoleCount:      1,
		},
		{
			name:         "successful invitation with multiple roles",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email: "multi_role@example.com",
				Name:  "Multi Role User",
				RoleIDs: []string{
					"e0000000-0000-0000-0000-000000000001", // Alpha admin role
					"e0000000-0000-0000-0000-000000000002", // Alpha editor role
				},
			},
			expectedStatus:         http.StatusOK,
			validateRoleAssignment: true,
			expectedRoleCount:      2,
		},
		{
			name:         "successful invitation with admin role",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequestBody{
				Email:   "new_admin@example.com",
				Name:    "New Admin User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"}, // Alpha admin role
			},
			expectedStatus:         http.StatusOK,
			validateRoleAssignment: true,
			expectedRoleCount:      1,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err, "Failed to marshal request body for test: %s", tt.name)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/users/invite", setup.BaseURL), bytes.NewBuffer(payloadBytes))
			assert.NoError(t, err, "Failed to create request for test: %s", tt.name)
			req.Header.Set("Content-Type", "application/json")

			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s for test: %s", tt.loginUserKey, tt.name)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request for test: %s", tt.name)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s. Body: %s - expected: %d, actual: %d", tt.name, string(payloadBytes), tt.expectedStatus, resp.StatusCode)

			if (tt.expectedStatus == http.StatusConflict || tt.expectedStatus == http.StatusBadRequest) && tt.expectedErrorKey != "" {
				var errResp ErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&errResp)
				assert.NoError(t, err, "Failed to decode error response for test: %s", tt.name)

				expectedMsg, msgOk := expectedInviteErrorMessages[tt.expectedErrorKey]
				assert.True(t, msgOk, "Expected error key %s not found in error messages map for test: %s", tt.expectedErrorKey, tt.name)
				assert.Equal(t, expectedMsg, errResp.Error, "Error message mismatch for test: %s", tt.name)
			}

			if tt.expectedStatus == http.StatusOK && tt.validateRoleAssignment {
				validateUserRoleAssignment(t, tt.requestBody.Email, tt.expectedRoleCount)
			}
		})
	}
}

func validateUserRoleAssignment(t *testing.T, email string, expectedRoleCount int) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	query := `
		SELECT COUNT(*)
		FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
		WHERE u.email = $1
	`

	var actualRoleCount int
	err := pool.QueryRow(context.Background(), query, email).Scan(&actualRoleCount)
	assert.NoError(t, err, "Failed to query role assignments for user %s", email)

	assert.Equal(t, expectedRoleCount, actualRoleCount,
		"Role count mismatch for user %s: expected %d, got %d",
		email, expectedRoleCount, actualRoleCount)
}
