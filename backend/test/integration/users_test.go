package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"dislyze/handlers"
	"dislyze/test/integration/setup"
)

func TestGetUsers_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name               string
		loginUserKey       string // Key for setup.TestUsersData map
		expectedStatus     int
		expectedUserEmails []string
		expectUnauth       bool
	}{
		{
			name:           "unauthenticated user gets 401",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "alpha_admin (Tenant A) gets users from Tenant Alpha",
			loginUserKey:   "alpha_admin",
			expectedStatus: http.StatusOK,
			// Order by created_at ASC from setup.sql: alpha_admin, alpha_editor, pending_editor_valid_token
			expectedUserEmails: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:           "alpha_editor (Tenant A) gets users from Tenant Alpha",
			loginUserKey:   "alpha_editor",
			expectedStatus: http.StatusOK,
			expectedUserEmails: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:               "beta_admin (Tenant B) gets users from Tenant Beta (only self)",
			loginUserKey:       "beta_admin",
			expectedStatus:     http.StatusOK,
			expectedUserEmails: []string{setup.TestUsersData["beta_admin"].Email},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/users", setup.BaseURL), nil)
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
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse []handlers.User
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				assert.Equal(t, len(tt.expectedUserEmails), len(usersResponse), "Number of users mismatch for test: %s", tt.name)

				actualEmails := make([]string, len(usersResponse))
				for i, u := range usersResponse {
					actualEmails[i] = u.Email
					assert.NotEmpty(t, u.ID, "User ID should not be empty for user %s", u.Email)

					var expectedName, expectedRole, expectedUserID, expectedStatus string
					foundInTestData := false
					for _, seededUser := range setup.TestUsersData {
						if seededUser.Email == u.Email {
							expectedName = seededUser.Name
							expectedRole = seededUser.Role
							expectedUserID = seededUser.UserID
							if u.Email == setup.TestUsersData["pending_editor_valid_token"].Email {
								expectedStatus = "pending_verification"
							} else {
								expectedStatus = "active"
							}
							foundInTestData = true
							break
						}
					}
					assert.True(t, foundInTestData, "User with email %s not found in setup.TestUsersData. Check setup.sql and setup.TestUsersData map.", u.Email)
					assert.Equal(t, expectedUserID, u.ID, "ID mismatch for user %s", u.Email)
					assert.Equal(t, expectedName, u.Name, "Name mismatch for user %s", u.Email)
					assert.Equal(t, expectedRole, u.Role, "Role mismatch for user %s", u.Email)
					assert.Equal(t, expectedStatus, u.Status, "Status mismatch for user %s", u.Email)
				}
				assert.Equal(t, tt.expectedUserEmails, actualEmails, "User email list or order mismatch for test: %s", tt.name)
			}
		})
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var expectedInviteErrorMessages = map[string]string{
	"emailConflict": "このメールアドレスは既に使用されています。",
}

func TestInviteUser_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)

	type inviteUserTestCase struct {
		name             string
		loginUserKey     string // Key for setup.TestUsersData map, empty for unauth
		requestBody      handlers.InviteUserRequest
		expectedStatus   int
		expectedErrorKey string // Key for expectedInviteErrorMessages, if any
		expectUnauth     bool
	}

	tests := []inviteUserTestCase{
		{
			name:         "successful invitation by alpha_admin",
			loginUserKey: "alpha_admin",
			requestBody: handlers.InviteUserRequest{
				Email: "new_invitee@example.com",
				Name:  "New Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:         "error when email already exists (alpha_admin invites existing alpha_editor)",
			loginUserKey: "alpha_admin",
			requestBody: handlers.InviteUserRequest{
				Email: setup.TestUsersData["alpha_editor"].Email,
				Name:  "Duplicate Invitee",
				Role:  "editor",
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorKey: "emailConflict",
		},
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			requestBody: handlers.InviteUserRequest{
				Email: "unauth_invitee@example.com",
				Name:  "Unauth Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "validation error: missing email",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid email format",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "invalid-email", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing name",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: name with only whitespace",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "whitespace@example.com", Name: "   ", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing role",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid role value",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: "guest"},
			expectedStatus: http.StatusBadRequest,
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
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s. Body: %s - expected: %d, actual: %d", tt.name, string(payloadBytes), tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusConflict && tt.expectedErrorKey != "" {
				var errResp ErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&errResp)
				assert.NoError(t, err, "Failed to decode error response for test: %s", tt.name)

				expectedMsg, msgOk := expectedInviteErrorMessages[tt.expectedErrorKey]
				assert.True(t, msgOk, "Expected error key %s not found in error messages map for test: %s", tt.expectedErrorKey, tt.name)
				assert.Equal(t, expectedMsg, errResp.Error, "Error message mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestAcceptInvite_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)

	const (
		plainValidTokenForAccept       = "accept-invite-plain-valid-token-for-testing-123"
		plainNonExistentTokenForAccept = "accept-invite-plain-nonexistent-token-for-testing-456"
		plainExpiredTokenForAccept     = "accept-invite-plain-expired-token-for-testing-789"
		plainTokenForActiveUserAccept  = "accept-invite-plain-token-for-active-user-000"
		newPasswordForAcceptInvite     = "SuP3rS3cur3N3wP@sswOrd!"
	)

	type acceptInviteTestCase struct {
		name           string
		requestBody    handlers.AcceptInviteRequest
		expectedStatus int
	}

	tests := []acceptInviteTestCase{
		{
			name: "successful invite acceptance",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "token not found",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainNonExistentTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "validation error - password mismatch",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept, // Needs a valid token context for this to be the failure point
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: "IncorrectP@sswOrdConfirm",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - password too short",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - empty token",
			requestBody: handlers.AcceptInviteRequest{
				Token:           "",
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "expired token",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainExpiredTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user status not pending_verification (e.g., already active)",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainTokenForActiveUserAccept, // Token associated with an already active user
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err, "Test: %s, Failed to marshal request body", tt.name)

			reqURL := fmt.Sprintf("%s/auth/accept-invite", setup.BaseURL)
			req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(payloadBytes))
			assert.NoError(t, err, "Test: %s, Failed to create request", tt.name)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			assert.NoError(t, err, "Test: %s, Failed to execute request", tt.name)
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Test: %s, Expected status %d, got %d. Body: %s", tt.name, tt.expectedStatus, resp.StatusCode, string(bodyBytes))
		})
	}
}
