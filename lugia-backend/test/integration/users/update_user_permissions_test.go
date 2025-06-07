package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateUserPermissions_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                 string
		loginUserKey         string // Key for setup.TestUsersData map, empty for unauth
		targetUserKey        string // Key for setup.TestUsersData map of target user
		targetUserIDOverride string // Use this if targetUserKey is empty (for invalid userID tests)
		requestBody          users.UpdateUserRoleRequestBody
		expectedStatus       int
		expectUnauth         bool
		validateResponse     func(t *testing.T, resp *http.Response) // For custom response validation
	}{
		// Authentication & Authorization Tests
		{
			name:           "unauthenticated request gets 401",
			targetUserKey:  "alpha_editor",
			requestBody:    users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "non-admin user gets 403 forbidden",
			loginUserKey:   "alpha_editor",
			targetUserKey:  "pending_editor_valid_token",
			requestBody:    users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin from different tenant gets 403 forbidden",
			loginUserKey:   "beta_admin",   // Tenant B
			targetUserKey:  "alpha_editor", // Tenant A
			requestBody:    users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:                 "invalid userID format gets 400",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "not-a-uuid",
			requestBody:          users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus:       http.StatusBadRequest,
		},
		{
			name:           "empty role gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    users.UpdateUserRoleRequestBody{Role: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role value gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    users.UpdateUserRoleRequestBody{Role: "guest"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON request gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic Tests
		{
			name:                 "non-existent user gets 404",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "00000000-0000-0000-0000-000000000000", // Valid UUID that doesn't exist
			requestBody:          users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus:       http.StatusNotFound,
		},
		{
			name:           "user trying to update own role gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_admin", // Same user
			requestBody:    users.UpdateUserRoleRequestBody{Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},

		// Success Tests
		{
			name:           "admin successfully updates editor to admin",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    users.UpdateUserRoleRequestBody{Role: "admin"},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
		{
			name:           "admin successfully updates admin to editor",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor", // Was updated to admin in previous test
			requestBody:    users.UpdateUserRoleRequestBody{Role: "editor"},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetUserID string
			if tt.targetUserKey != "" {
				targetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				assert.True(t, ok, "Target user key '%s' not found in setup.TestUsersData", tt.targetUserKey)
				targetUserID = targetUserDetails.UserID
			} else if tt.targetUserIDOverride != "" {
				targetUserID = tt.targetUserIDOverride
			} else {
				t.Fatal("Either targetUserKey or targetUserIDOverride must be provided")
			}

			var reqBody []byte
			var err error

			if tt.name == "malformed JSON request gets 400" {
				// Send malformed JSON for this specific test
				reqBody = []byte(`{"role": "admin", invalid}`)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err, "Failed to marshal request body")
			}

			reqURL := fmt.Sprintf("%s/users/%s/permissions", setup.BaseURL, targetUserID)
			req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqBody))
			assert.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Add authentication if not testing unauthenticated scenario
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key '%s' not found in setup.TestUsersData", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request")
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			// Custom response validation if provided
			if tt.validateResponse != nil {
				tt.validateResponse(t, resp)
			}

			// For successful updates, verify the role was actually changed in database
			if tt.expectedStatus == http.StatusOK && tt.targetUserKey != "" {
				ctx := context.Background()
				var actualRole string
				err = pool.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", targetUserID).Scan(&actualRole)
				assert.NoError(t, err, "Failed to query updated user role from database")

				expectedRole := strings.TrimSpace(strings.ToLower(string(tt.requestBody.Role)))
				assert.Equal(t, expectedRole, actualRole, "Role was not updated correctly in database")
			}
		})
	}
}
