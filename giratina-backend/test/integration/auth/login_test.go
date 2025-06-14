package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"giratina/features/auth"
	"giratina/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	testUser := setup.TestUsersData["internal_1"]
	nonInternalUser := setup.TestUsersData["enterprise_1"]

	tests := []struct {
		name           string
		request        auth.LoginRequestBody
		expectedStatus int
	}{
		{
			name: "successful login for internal admin",
			request: auth.LoginRequestBody{
				Email:    testUser.Email,
				Password: testUser.PlainTextPassword,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid user but not internal admin",
			request: auth.LoginRequestBody{
				Email:    nonInternalUser.Email,
				Password: nonInternalUser.PlainTextPassword,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong password",
			request: auth.LoginRequestBody{
				Email:    testUser.Email,
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "non-existent email",
			request: auth.LoginRequestBody{
				Email:    "nonexistent@example.com",
				Password: testUser.PlainTextPassword,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing email",
			request: auth.LoginRequestBody{
				Password: testUser.PlainTextPassword,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: auth.LoginRequestBody{
				Email: testUser.Email,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {

				cookies := resp.Cookies()
				assert.NotEmpty(t, cookies, "Expected cookies in response for successful login")

				var accessToken, refreshToken *http.Cookie
				for _, cookie := range cookies {
					switch cookie.Name {
					case "dislyze_access_token":
						accessToken = cookie
					case "dislyze_refresh_token":
						refreshToken = cookie
					}
				}

				assert.NotNil(t, accessToken, "Access token cookie not found")
				assert.True(t, accessToken.HttpOnly, "Access token cookie should be HttpOnly")
				assert.True(t, accessToken.Secure, "Access token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, accessToken.SameSite, "Access token cookie should have SameSite=Strict")

				assert.NotNil(t, refreshToken, "Refresh token cookie not found")
				assert.True(t, refreshToken.HttpOnly, "Refresh token cookie should be HttpOnly")
				assert.True(t, refreshToken.Secure, "Refresh token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, refreshToken.SameSite, "Refresh token cookie should have SameSite=Strict")
			} else {
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed login with status %d", tt.expectedStatus)
			}
		})
	}
}
