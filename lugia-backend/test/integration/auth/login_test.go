package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/features/auth"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	testUser := setup.TestUsersData["enterprise_1"]

	tests := []struct {
		name           string
		request        auth.LoginRequestBody
		expectedStatus int
	}{
		{
			name: "successful login",
			request: auth.LoginRequestBody{
				Email:    testUser.Email,
				Password: testUser.PlainTextPassword,
			},
			expectedStatus: http.StatusOK,
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

func TestLoginLogoutAndVerifyMeEndpoint(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	testUser := setup.TestUsersData["enterprise_1"]
	client := &http.Client{}

	// 1. Log in
	loginPayload := auth.LoginRequestBody{
		Email:    testUser.Email,
		Password: testUser.PlainTextPassword,
	}
	loginBody, err := json.Marshal(loginPayload)
	assert.NoError(t, err)

	loginReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(loginBody))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer func() {
		if err := loginResp.Body.Close(); err != nil {
			t.Logf("Error closing loginResp body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, loginResp.StatusCode, "Login request failed")

	loginCookies := loginResp.Cookies()
	assert.NotEmpty(t, loginCookies, "Expected cookies from successful login")

	// 2. Call /me and confirm 200 OK (when logged in)
	meReqLoggedIn, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)
	for _, cookie := range loginCookies {
		meReqLoggedIn.AddCookie(cookie)
	}

	meRespLoggedIn, err := client.Do(meReqLoggedIn)
	assert.NoError(t, err)
	defer func() {
		if err := meRespLoggedIn.Body.Close(); err != nil {
			t.Logf("Error closing meRespLoggedIn body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, meRespLoggedIn.StatusCode, "/me endpoint should return 200 OK when logged in")

	// 3. Log out
	logoutReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/logout", setup.BaseURL), nil)
	assert.NoError(t, err)
	for _, cookie := range loginCookies {
		logoutReq.AddCookie(cookie)
	}

	logoutResp, err := client.Do(logoutReq)
	assert.NoError(t, err)
	defer func() {
		if err := logoutResp.Body.Close(); err != nil {
			t.Logf("Error closing logoutResp body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, logoutResp.StatusCode, "Logout request should return 200 OK")

	var accessTokenCleared, refreshTokenCleared bool
	for _, cookie := range logoutResp.Cookies() {
		if cookie.Name == "dislyze_access_token" {
			assert.Equal(t, -1, cookie.MaxAge, "Access token cookie should be cleared")
			accessTokenCleared = true
		}
		if cookie.Name == "dislyze_refresh_token" {
			assert.Equal(t, -1, cookie.MaxAge, "Refresh token cookie should be cleared")
			refreshTokenCleared = true
		}
	}
	assert.True(t, accessTokenCleared, "Access token clear instruction not found in Set-Cookie header")
	assert.True(t, refreshTokenCleared, "Refresh token clear instruction not found in Set-Cookie header")

	// 4. Call /me and confirm 401 Unauthorized (after logout, no cookies sent)
	meReqLoggedOut, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)

	meRespLoggedOut, err := client.Do(meReqLoggedOut)
	assert.NoError(t, err)
	defer func() {
		if err := meRespLoggedOut.Body.Close(); err != nil {
			t.Logf("Error closing meRespLoggedOut body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusUnauthorized, meRespLoggedOut.StatusCode, "/me endpoint should return 401 Unauthorized after logout and no cookies sent")
}
