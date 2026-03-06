package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lugia/features/auth"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type loginErrorResponse struct {
	Error string `json:"error"`
}

func TestLoginUserStatus_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                 string
		loginUserKey         string
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name:                 "suspended user cannot login",
			loginUserKey:         "enterprise_16",
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "アカウントが停止されています。サポートにお問い合わせください。",
		},
		{
			name:                 "another suspended user cannot login",
			loginUserKey:         "enterprise_17",
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "アカウントが停止されています。サポートにお問い合わせください。",
		},
		{
			name:                 "pending verification user cannot login",
			loginUserKey:         "enterprise_11",
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "アカウントが有効化されていません。招待メールを確認し、登録を完了してください。",
		},
		{
			name:                 "another pending verification user cannot login",
			loginUserKey:         "enterprise_12",
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "アカウントが有効化されていません。招待メールを確認し、登録を完了してください。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := setup.TestUsersData[tt.loginUserKey]

			body, err := json.Marshal(auth.LoginRequestBody{
				Email:    userData.Email,
				Password: userData.PlainTextPassword,
			})
			require.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Verify no auth cookies are set
			cookies := resp.Cookies()
			for _, cookie := range cookies {
				assert.NotEqual(t, "dislyze_access_token", cookie.Name, "Access token should not be set for %s user", tt.loginUserKey)
				assert.NotEqual(t, "dislyze_refresh_token", cookie.Name, "Refresh token should not be set for %s user", tt.loginUserKey)
			}

			// Verify error message
			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var errResp loginErrorResponse
			err = json.Unmarshal(bodyBytes, &errResp)
			require.NoError(t, err, "Response body should be valid JSON: %s", string(bodyBytes))
			assert.Equal(t, tt.expectedErrorMessage, errResp.Error)
		})
	}
}

func TestLoginSSOOnlyTenant_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	// sso_1 is an active user in the SSO-only tenant (auth_method = 'sso')
	ssoUser := setup.TestUsersData["sso_1"]

	body, err := json.Marshal(auth.LoginRequestBody{
		Email:    ssoUser.Email,
		Password: ssoUser.PlainTextPassword,
	})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Verify no auth cookies are set
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		assert.NotEqual(t, "dislyze_access_token", cookie.Name, "Access token should not be set for SSO user")
		assert.NotEqual(t, "dislyze_refresh_token", cookie.Name, "Refresh token should not be set for SSO user")
	}

	// Verify correct error message
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errResp loginErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	require.NoError(t, err, "Response body should be valid JSON: %s", string(bodyBytes))
	assert.Equal(t, "このアカウントはSSO専用です。SSOでログインしてください。", errResp.Error)
}

func TestLoginErrorMessages_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                 string
		request              auth.LoginRequestBody
		expectedStatus       int
		expectedErrorMessage string
	}{
		{
			name: "wrong password returns generic credential error",
			request: auth.LoginRequestBody{
				Email:    setup.TestUsersData["enterprise_1"].Email,
				Password: "wrongpassword",
			},
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "メールアドレスまたはパスワードが正しくありません",
		},
		{
			name: "non-existent email returns same generic credential error",
			request: auth.LoginRequestBody{
				Email:    "nonexistent@example.com",
				Password: "somepassword",
			},
			expectedStatus:       http.StatusUnauthorized,
			expectedErrorMessage: "メールアドレスまたはパスワードが正しくありません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var errResp loginErrorResponse
			err = json.Unmarshal(bodyBytes, &errResp)
			require.NoError(t, err, "Response body should be valid JSON: %s", string(bodyBytes))

			// Both wrong password and non-existent email should return the same message
			// to prevent user enumeration attacks
			assert.Equal(t, tt.expectedErrorMessage, errResp.Error)
		})
	}
}

func TestLoginInvalidRequestBody_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid JSON returns 400",
			body:           `{"email": "test@example.com", "password":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty JSON object returns 400",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty string body returns 400",
			body:           ``,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "whitespace-only email returns 400",
			body:           `{"email": "   ", "password": "somepassword"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "whitespace-only password returns 400",
			body:           `{"email": "test@example.com", "password": "   "}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer([]byte(tt.body)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// No auth cookies should be set on failure
			cookies := resp.Cookies()
			for _, cookie := range cookies {
				assert.NotEqual(t, "dislyze_access_token", cookie.Name, "Access token should not be set on invalid request")
				assert.NotEqual(t, "dislyze_refresh_token", cookie.Name, "Refresh token should not be set on invalid request")
			}
		})
	}
}

func TestLoginRefreshTokenCreation_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	testUser := setup.TestUsersData["enterprise_5"]

	// Count refresh tokens before login
	var beforeCount int
	err := pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1", testUser.UserID).Scan(&beforeCount)
	require.NoError(t, err)

	// Perform login
	body, err := json.Marshal(auth.LoginRequestBody{
		Email:    testUser.Email,
		Password: testUser.PlainTextPassword,
	})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Count refresh tokens after login - should have at least one
	var afterCount int
	err = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1", testUser.UserID).Scan(&afterCount)
	require.NoError(t, err)
	assert.Greater(t, afterCount, 0, "Successful login should create a refresh token in the database")
}

func TestLoginFailedDoesNotCreateRefreshToken_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	suspendedUser := setup.TestUsersData["enterprise_16"]

	// Count refresh tokens before failed login attempt
	var beforeCount int
	err := pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1", suspendedUser.UserID).Scan(&beforeCount)
	require.NoError(t, err)

	// Attempt login with suspended user
	body, err := json.Marshal(auth.LoginRequestBody{
		Email:    suspendedUser.Email,
		Password: suspendedUser.PlainTextPassword,
	})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Count refresh tokens after failed login - should not have increased
	var afterCount int
	err = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1", suspendedUser.UserID).Scan(&afterCount)
	require.NoError(t, err)
	assert.Equal(t, beforeCount, afterCount, "Failed login should not create a refresh token")
}

func TestLoginCrossTenant_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	// Verify users from different tenants can all login independently
	tests := []struct {
		name         string
		loginUserKey string
	}{
		{
			name:         "enterprise tenant user can login",
			loginUserKey: "enterprise_1",
		},
		{
			name:         "SMB tenant user can login",
			loginUserKey: "smb_1",
		},
		{
			name:         "internal tenant user can login",
			loginUserKey: "internal_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := setup.TestUsersData[tt.loginUserKey]

			body, err := json.Marshal(auth.LoginRequestBody{
				Email:    userData.Email,
				Password: userData.PlainTextPassword,
			})
			require.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "User from %s should be able to login", tt.loginUserKey)

			// Verify cookies are set with correct security attributes
			var accessTokenFound, refreshTokenFound bool
			for _, cookie := range resp.Cookies() {
				switch cookie.Name {
				case "dislyze_access_token":
					accessTokenFound = true
					assert.NotEmpty(t, cookie.Value)
					assert.True(t, cookie.HttpOnly)
					assert.Equal(t, "/", cookie.Path)
				case "dislyze_refresh_token":
					refreshTokenFound = true
					assert.NotEmpty(t, cookie.Value)
					assert.True(t, cookie.HttpOnly)
					assert.Equal(t, "/", cookie.Path)
					assert.Equal(t, 7*24*60*60, cookie.MaxAge, "Refresh token should expire in 7 days")
				}
			}
			assert.True(t, accessTokenFound, "Access token cookie should be set")
			assert.True(t, refreshTokenFound, "Refresh token cookie should be set")
		})
	}
}
