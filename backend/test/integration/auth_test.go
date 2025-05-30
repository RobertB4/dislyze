package integration

import (
	"bytes"
	"dislyze/test/integration/setup"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SignupRequest struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

type SignupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func TestSignup(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		request        SignupRequest
		expectedStatus int
	}{
		{
			name: "successful signup",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing company name",
			request: SignupRequest{
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing user name",
			request: SignupRequest{
				CompanyName:     "Test Company",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing email",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: SignupRequest{
				CompanyName: "Test Company",
				UserName:    "Test User",
				Email:       "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "passwords do not match",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", setup.BaseURL), bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response SignupResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.True(t, response.Success, "Expected success to be true for OK status")
				assert.Empty(t, response.Error, "Expected error to be empty for OK status")

				cookies := resp.Cookies()
				assert.NotEmpty(t, cookies, "Expected cookies in response for successful signup")

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
			} else if tt.expectedStatus == http.StatusBadRequest {
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed signup with status %d", tt.expectedStatus)
			} else {
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed signup with status %d", tt.expectedStatus)
			}
		})
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	request := SignupRequest{
		CompanyName:     "Test Company",
		UserName:        "Test User",
		Email:           "duplicate@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}

	body, err := json.Marshal(request)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", setup.BaseURL), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var successResponse SignupResponse
	err = json.NewDecoder(resp.Body).Decode(&successResponse)
	assert.NoError(t, err)
	assert.True(t, successResponse.Success)
	assert.Empty(t, successResponse.Error)

	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies, "Expected cookies for initial successful signup")
	var accessToken, refreshToken *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "dislyze_access_token":
			accessToken = cookie
		case "dislyze_refresh_token":
			refreshToken = cookie
		}
	}
	assert.NotNil(t, accessToken, "Access token cookie not found (initial signup)")
	assert.NotNil(t, refreshToken, "Refresh token cookie not found (initial signup)")

	req2, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", setup.BaseURL), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var errorResponse SignupResponse
	err = json.NewDecoder(resp2.Body).Decode(&errorResponse)
	assert.NoError(t, err)
	assert.False(t, errorResponse.Success)
	assert.Equal(t, "このメールアドレスは既に使用されています。", errorResponse.Error)

	// Expect no cookies for the failed duplicate email signup
	cookies2 := resp2.Cookies()
	assert.Empty(t, cookies2, "Expected no cookies for duplicate email signup attempt")
}

func TestLogin(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	createTestUser(t)

	tests := []struct {
		name           string
		request        setup.LoginRequest
		expectedStatus int
	}{
		{
			name: "successful login",
			request: setup.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			request: setup.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "non-existent email",
			request: setup.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing email",
			request: setup.LoginRequest{
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: setup.LoginRequest{
				Email: "test@example.com",
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
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var response LoginResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.True(t, response.Success, "Expected success to be true for OK status")
				assert.Empty(t, response.Error, "Expected error to be empty for OK status")

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

func createTestUser(t *testing.T) {
	body, err := json.Marshal(SignupRequest{
		CompanyName:     "Test Company",
		UserName:        "Test User",
		Email:           "test@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	})
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", setup.BaseURL), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLoginLogoutAndVerifyMeEndpoint(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	createTestUser(t)

	client := &http.Client{}

	// 1. Log in
	loginPayload := setup.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginBody, err := json.Marshal(loginPayload)
	assert.NoError(t, err)

	loginReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(loginBody))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()
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
	defer meRespLoggedIn.Body.Close()
	assert.Equal(t, http.StatusOK, meRespLoggedIn.StatusCode, "/me endpoint should return 200 OK when logged in")

	// 3. Log out
	logoutReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/logout", setup.BaseURL), nil)
	assert.NoError(t, err)
	for _, cookie := range loginCookies {
		logoutReq.AddCookie(cookie)
	}

	logoutResp, err := client.Do(logoutReq)
	assert.NoError(t, err)
	defer logoutResp.Body.Close()

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
	defer meRespLoggedOut.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, meRespLoggedOut.StatusCode, "/me endpoint should return 401 Unauthorized after logout and no cookies sent")
}
