package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateMe_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map, empty for unauth
		requestBody    users.UpdateMeRequestBody
		customJSON     string // For malformed JSON tests
		expectedStatus int
		expectUnauth   bool
	}{
		// Authentication Tests
		{
			name:           "unauthenticated request gets 401",
			requestBody:    users.UpdateMeRequestBody{Name: "Test Name"},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},

		// Input Validation Tests
		{
			name:           "empty name gets 400",
			loginUserKey:   "enterprise_1",
			requestBody:    users.UpdateMeRequestBody{Name: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "name with only whitespace gets 400",
			loginUserKey:   "enterprise_1",
			requestBody:    users.UpdateMeRequestBody{Name: "   "},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON request gets 400",
			loginUserKey:   "enterprise_1",
			customJSON:     `{"name": "Valid Name", invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing name field gets 400",
			loginUserKey:   "enterprise_1",
			customJSON:     `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "name exceeding 255 characters gets 500",
			loginUserKey:   "enterprise_1",
			requestBody:    users.UpdateMeRequestBody{Name: strings.Repeat("a", 256)}, // 256 chars, over limit
			expectedStatus: http.StatusInternalServerError,
		},

		// Success Tests
		{
			name:           "enterprise_1 successfully updates name",
			loginUserKey:   "enterprise_1",
			requestBody:    users.UpdateMeRequestBody{Name: "Updated Enterprise Admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "smb_1 successfully updates name",
			loginUserKey:   "smb_1",
			requestBody:    users.UpdateMeRequestBody{Name: "Updated SMB Admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "name with leading/trailing whitespace is trimmed",
			loginUserKey:   "enterprise_2",
			requestBody:    users.UpdateMeRequestBody{Name: "  Trimmed Name  "},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "name with special characters works",
			loginUserKey:   "enterprise_2",
			requestBody:    users.UpdateMeRequestBody{Name: "Jean-Claude O'Connor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "name with unicode characters works",
			loginUserKey:   "enterprise_2",
			requestBody:    users.UpdateMeRequestBody{Name: "田中太郎 🎉 Émilie"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "maximum length name (255 chars) works",
			loginUserKey:   "enterprise_2",
			requestBody:    users.UpdateMeRequestBody{Name: strings.Repeat("a", 255)}, // Exactly 255 chars
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if tt.customJSON != "" {
				// Use custom JSON for malformed JSON tests
				reqBody = []byte(tt.customJSON)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err, "Failed to marshal request body")
			}

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-name", setup.BaseURL), bytes.NewBuffer(reqBody))
			assert.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Add authentication cookies if not testing unauth
			if !tt.expectUnauth && tt.loginUserKey != "" {
				userDetails, ok := setup.TestUsersData2[tt.loginUserKey]
				assert.True(t, ok, "User key '%s' not found in setup.TestUsersData", tt.loginUserKey)

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, userDetails.Email, userDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
				req.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request")
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)
		})
	}
}
