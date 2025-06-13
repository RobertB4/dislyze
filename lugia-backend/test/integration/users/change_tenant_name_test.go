package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangeTenantName_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string
		requestBody    interface{}
		expectedStatus int
		expectUnauth   bool
	}{
		// Authentication & Authorization
		{
			name:           "unauthenticated request returns 401",
			requestBody:    map[string]string{"name": "New Tenant Name"},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "editor role gets 403 (not admin)",
			loginUserKey:   "enterprise_2",
			requestBody:    map[string]string{"name": "New Tenant Name"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin role gets 200 (authorized)",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{"name": "New Tenant Name"},
			expectedStatus: http.StatusOK,
		},

		// Request Validation
		{
			name:           "empty request body returns 400",
			loginUserKey:   "enterprise_1",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON returns 400",
			loginUserKey:   "enterprise_1",
			requestBody:    `{"name": "unclosed string`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing name field returns 400",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty name string returns 400",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{"name": ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "name with only whitespace returns 400",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{"name": "   \t\n   "},
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic
		{
			name:           "successful update with no response body",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{"name": "Updated Company Name"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "name too long (>255 chars) returns 500",
			loginUserKey:   "enterprise_1",
			requestBody:    map[string]string{"name": strings.Repeat("a", 256)},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody io.Reader

			// Handle different request body types
			switch body := tt.requestBody.(type) {
			case string:
				if body == "" {
					reqBody = strings.NewReader("")
				} else {
					reqBody = strings.NewReader(body)
				}
			case map[string]string:
				jsonBody, err := json.Marshal(body)
				assert.NoError(t, err)
				reqBody = bytes.NewBuffer(jsonBody)
			default:
				jsonBody, err := json.Marshal(body)
				assert.NoError(t, err)
				reqBody = bytes.NewBuffer(jsonBody)
			}

			reqURL := fmt.Sprintf("%s/tenant/change-name", setup.BaseURL)
			req, err := http.NewRequest("POST", reqURL, reqBody)
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData2[tt.loginUserKey]
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

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

		})
	}
}
