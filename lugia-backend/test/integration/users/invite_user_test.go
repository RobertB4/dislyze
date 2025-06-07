package users

import (
	"bytes"
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
}

func TestInviteUser_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	type inviteUserTestCase struct {
		name             string
		loginUserKey     string // Key for setup.TestUsersData map, empty for unauth
		requestBody      users.InviteUserRequest
		expectedStatus   int
		expectedErrorKey string // Key for expectedInviteErrorMessages, if any
		expectUnauth     bool
	}

	tests := []inviteUserTestCase{
		{
			name:         "successful invitation by alpha_admin",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequest{
				Email: "new_invitee@example.com",
				Name:  "New Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "error when email already exists (alpha_admin invites existing alpha_editor)",
			loginUserKey: "alpha_admin",
			requestBody: users.InviteUserRequest{
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
			requestBody: users.InviteUserRequest{
				Email: "unauth_invitee@example.com",
				Name:  "Unauth Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "validation error: missing email",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid email format",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "invalid-email", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing name",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "valid@example.com", Name: "", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: name with only whitespace",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "whitespace@example.com", Name: "   ", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing role",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid role value",
			loginUserKey:   "alpha_admin",
			requestBody:    users.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: "guest"},
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
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

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
