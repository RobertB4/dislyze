package integration

import (
	"bytes"
	"context"
	"dislyze/test/integration/setup"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}
type ForgotPasswordResponse struct {
	Success bool `json:"success"`
}

type VerifyResetTokenRequest struct {
	Token string `json:"token"`
}

type VerifyResetTokenResponse struct {
	Success bool   `json:"success"`
	Email   string `json:"email,omitempty"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

// ResetPasswordResponse defines the structure for the reset password response body.
// Success: Indicates whether the password reset operation was successful.
type ResetPasswordResponse struct {
	Success bool `json:"success"`
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
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

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
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Logf("Error closing response body for resp2: %v", err)
		}
	}()

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
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

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

type SendgridMockEmailContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SendgridMockTo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type SendgridMockPersonalization struct {
	To      []SendgridMockTo `json:"to"`
	Subject string           `json:"subject"`
}

type SendgridMockEmail struct {
	Personalizations []SendgridMockPersonalization `json:"personalizations"`
	Content          []SendgridMockEmailContent    `json:"content"`
	SentAt           int64                         `json:"sent_at"`
}

func getLatestEmailFromSendgridMock(t *testing.T, expectedRecipientEmail string) (*SendgridMockEmail, error) {
	t.Helper()
	sendgridAPIURL := os.Getenv("SENDGRID_API_URL")
	sendgridAPIKey := os.Getenv("SENDGRID_API_KEY")

	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error

	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/json?token=%s", sendgridAPIURL, sendgridAPIKey), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to sendgrid-mock: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get emails from sendgrid-mock: %w", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body in getLatestEmailFromSendgridMock: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("sendgrid-mock returned status %d", resp.StatusCode)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var emails []SendgridMockEmail
		if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
			lastErr = fmt.Errorf("failed to decode emails from sendgrid-mock: %w", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(emails) > 0 {
			latestEmail := emails[0]
			if len(latestEmail.Personalizations) > 0 && len(latestEmail.Personalizations[0].To) > 0 &&
				latestEmail.Personalizations[0].To[0].Email == expectedRecipientEmail {
				return &latestEmail, nil
			}
			lastErr = fmt.Errorf("latest email recipient %s does not match expected %s", latestEmail.Personalizations[0].To[0].Email, expectedRecipientEmail)
		} else {
			lastErr = fmt.Errorf("no emails found in sendgrid-mock")
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("failed to get expected email for %s after multiple retries: %w", expectedRecipientEmail, lastErr)
}

func extractResetTokenFromEmail(t *testing.T, email *SendgridMockEmail) (string, error) {
	t.Helper()
	for _, content := range email.Content {
		if content.Type == "text/html" {
			re := regexp.MustCompile(`href="[^"]*/reset-password\?token=([a-zA-Z0-9\-_.%]+)"`)
			matches := re.FindStringSubmatch(content.Value)
			if len(matches) > 1 {
				decodedToken, err := url.QueryUnescape(matches[1])
				if err != nil {
					return "", fmt.Errorf("failed to decode reset token from email: %w", err)
				}
				return decodedToken, nil
			}
		}
	}
	return "", fmt.Errorf("reset token not found in email HTML content")
}

func attemptLogin(t *testing.T, email string, password string) *http.Response {
	t.Helper()
	client := &http.Client{}

	loginPayload := setup.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, err := json.Marshal(loginPayload)
	assert.NoError(t, err, "Failed to marshal login payload in attemptLogin")

	loginReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(loginBody))
	assert.NoError(t, err, "Failed to create login request in attemptLogin")
	loginReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(loginReq)
	assert.NoError(t, err, "Failed to execute login request in attemptLogin")
	return resp
}

func TestForgotPassword(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	t.Run("TestForgotPassword_ExistingEmail_Successful", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_admin"]
		payload := ForgotPasswordRequest{Email: testUser.Email}
		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var apiResp ForgotPasswordResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.True(t, apiResp.Success)

		email, err := getLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err, "Failed to get email from SendGrid mock")
		if err == nil {
			assert.Equal(t, "パスワードリセットのご案内 - dislyze", email.Personalizations[0].Subject)

			rawToken, err := extractResetTokenFromEmail(t, email)
			assert.NoError(t, err, "Failed to extract reset token from email")
			assert.NotEmpty(t, rawToken, "Extracted reset token should not be empty")

			hash := sha256.Sum256([]byte(rawToken))
			hashedTokenStr := hex.EncodeToString(hash[:])

			var dbTokenHash string
			var dbUserID pgtype.UUID
			var dbExpiresAt pgtype.Timestamptz
			var dbUsedAt pgtype.Timestamptz // Should be NULL

			ctx := context.Background()
			dbErr := pool.QueryRow(ctx, "SELECT token_hash, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbTokenHash, &dbUserID, &dbExpiresAt, &dbUsedAt)
			assert.NoError(t, dbErr, "Failed to query password_reset_tokens table")

			if dbErr == nil {
				assert.Equal(t, hashedTokenStr, dbTokenHash)

				var expectedPgUUID pgtype.UUID
				err = expectedPgUUID.Scan(testUser.UserID)
				assert.NoError(t, err, "Failed to scan testUser.UserID into pgtype.UUID")
				assert.Equal(t, expectedPgUUID, dbUserID, "User ID in token record does not match")

				assert.True(t, dbExpiresAt.Time.After(time.Now()), "Token expiry should be in the future")
				assert.True(t, dbExpiresAt.Time.Before(time.Now().Add(35*time.Minute)), "Token expiry should be around 30 mins") // Check within a reasonable window
				assert.False(t, dbUsedAt.Valid, "Token should not be used yet")
			}
		}
	})

	t.Run("TestForgotPassword_NonExistentEmail", func(t *testing.T) {
		nonExistentEmail := "idonotexist@example.com"
		payload := ForgotPasswordRequest{Email: nonExistentEmail}
		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var apiResp ForgotPasswordResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.True(t, apiResp.Success)
	})

	t.Run("TestForgotPassword_InvalidEmailFormat", func(t *testing.T) {
		payload := ForgotPasswordRequest{Email: "invalidemail"}
		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp ForgotPasswordResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success, "Expected success to be false for invalid email format")
	})

	t.Run("TestForgotPassword_EmptyEmail", func(t *testing.T) {
		payload := ForgotPasswordRequest{Email: ""}
		body, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp ForgotPasswordResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success, "Expected success to be false for empty email")
	})

	t.Run("TestForgotPassword_MultipleRequestsForSameUser", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]

		// --- First Request ---
		payload1 := ForgotPasswordRequest{Email: testUser.Email}
		body1, err := json.Marshal(payload1)
		assert.NoError(t, err)
		req1, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body1))
		assert.NoError(t, err)
		req1.Header.Set("Content-Type", "application/json")
		resp1, err := client.Do(req1)
		assert.NoError(t, err)
		defer func() {
			if err := resp1.Body.Close(); err != nil {
				t.Logf("Error closing resp1 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		email1, err := getLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err)
		rawToken1, err := extractResetTokenFromEmail(t, email1)
		assert.NoError(t, err)
		assert.NotEmpty(t, rawToken1)
		hash1 := sha256.Sum256([]byte(rawToken1))
		hashedTokenStr1 := hex.EncodeToString(hash1[:])

		// Verify Token 1 in DB
		var dbTokenHash1 string
		var dbUserID1 pgtype.UUID
		var dbExpiresAt1 pgtype.Timestamptz
		var dbUsedAt1 pgtype.Timestamptz
		ctx := context.Background()
		err = pool.QueryRow(ctx, "SELECT token_hash, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr1).Scan(&dbTokenHash1, &dbUserID1, &dbExpiresAt1, &dbUsedAt1)
		assert.NoError(t, err, "Token 1 should exist in DB after first request")
		if err == nil {
			assert.Equal(t, hashedTokenStr1, dbTokenHash1)
			var expectedPgUUID1 pgtype.UUID
			scanErr := expectedPgUUID1.Scan(testUser.UserID)
			assert.NoError(t, scanErr)
			assert.Equal(t, expectedPgUUID1, dbUserID1)
			assert.True(t, dbExpiresAt1.Time.After(time.Now()))
			assert.False(t, dbUsedAt1.Valid)
		}

		// --- Second Request ---
		payload2 := ForgotPasswordRequest{Email: testUser.Email}
		body2, err := json.Marshal(payload2)
		assert.NoError(t, err)
		req2, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body2))
		assert.NoError(t, err)
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := client.Do(req2)
		assert.NoError(t, err)
		defer func() {
			if err := resp2.Body.Close(); err != nil {
				t.Logf("Error closing resp2 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		email2, err := getLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err)
		rawToken2, err := extractResetTokenFromEmail(t, email2)
		assert.NoError(t, err)
		assert.NotEmpty(t, rawToken2)
		assert.NotEqual(t, rawToken1, rawToken2, "Raw tokens from two requests should be different")
		hash2 := sha256.Sum256([]byte(rawToken2))
		hashedTokenStr2 := hex.EncodeToString(hash2[:])

		// Verify Token 1 is gone from DB
		var placeholder string // We don't care about the value, just if the row exists
		err = pool.QueryRow(ctx, "SELECT token_hash FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr1).Scan(&placeholder)
		assert.Error(t, err, "Token 1 should have been deleted or invalidated")
		assert.Equal(t, pgx.ErrNoRows, err, "Expected pgx.ErrNoRows when querying for Token 1")

		// Verify Token 2 in DB
		var dbTokenHash2 string
		var dbUserID2 pgtype.UUID
		var dbExpiresAt2 pgtype.Timestamptz
		var dbUsedAt2 pgtype.Timestamptz // Should be NULL

		err = pool.QueryRow(ctx, "SELECT token_hash, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr2).Scan(&dbTokenHash2, &dbUserID2, &dbExpiresAt2, &dbUsedAt2)
		assert.NoError(t, err, "Token 2 should exist in DB after second request")

		if err == nil {
			assert.Equal(t, hashedTokenStr2, dbTokenHash2)

			var expectedPgUUID2 pgtype.UUID
			scanErr := expectedPgUUID2.Scan(testUser.UserID)
			assert.NoError(t, scanErr, "Failed to scan testUser.UserID into pgtype.UUID for Token 2")
			assert.Equal(t, expectedPgUUID2, dbUserID2, "User ID in Token 2 record does not match")

			assert.True(t, dbExpiresAt2.Time.After(time.Now()), "Token 2 expiry should be in the future")
			assert.True(t, dbExpiresAt2.Time.Before(time.Now().Add(35*time.Minute)), "Token 2 expiry should be around 30 mins") // Check within a reasonable window
			assert.False(t, dbUsedAt2.Valid, "Token 2 should not be used yet")
		}
	})
}

func TestVerifyResetToken(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	ctx := context.Background()

	// Helper function to make a /auth/forgot-password request and get the raw token
	getRawResetToken := func(userEmail string) string {
		payload := ForgotPasswordRequest{Email: userEmail}
		body, err := json.Marshal(payload)
		assert.NoError(t, err)
		fpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err)
		fpReq.Header.Set("Content-Type", "application/json")
		fpResp, err := client.Do(fpReq)
		assert.NoError(t, err)
		defer func() {
			if err := fpResp.Body.Close(); err != nil {
				t.Logf("Error closing fpResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, fpResp.StatusCode)

		email, err := getLatestEmailFromSendgridMock(t, userEmail)
		assert.NoError(t, err)
		rawToken, err := extractResetTokenFromEmail(t, email)
		assert.NoError(t, err)
		assert.NotEmpty(t, rawToken)
		return rawToken
	}

	t.Run("TestVerifyResetToken_ValidToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_admin"]
		rawToken := getRawResetToken(testUser.Email)
		fmt.Println("Raw token for user:", testUser.Email, "is", rawToken)

		verifyPayload := VerifyResetTokenRequest{Token: rawToken}
		verifyBody, err := json.Marshal(verifyPayload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/verify-reset-token", setup.BaseURL), bytes.NewBuffer(verifyBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var apiResp VerifyResetTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.True(t, apiResp.Success)
		assert.Equal(t, testUser.Email, apiResp.Email)

		// Verify DB token is not marked as used
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist")
		assert.False(t, dbUsedAt.Valid, "Token should not be marked as used after verification")
	})

	t.Run("TestVerifyResetToken_InvalidToken_NonExistent", func(t *testing.T) {
		verifyPayload := VerifyResetTokenRequest{Token: "non-existent-token-string"}
		verifyBody, err := json.Marshal(verifyPayload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/verify-reset-token", setup.BaseURL), bytes.NewBuffer(verifyBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp VerifyResetTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.Empty(t, apiResp.Email)
	})

	t.Run("TestVerifyResetToken_ExpiredToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		rawToken := getRawResetToken(testUser.Email)

		// Manually expire the token in the DB
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET expires_at = $1 WHERE token_hash = $2", time.Now().Add(-1*time.Hour), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually expire token")

		verifyPayload := VerifyResetTokenRequest{Token: rawToken}
		verifyBody, err := json.Marshal(verifyPayload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/verify-reset-token", setup.BaseURL), bytes.NewBuffer(verifyBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp VerifyResetTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.Empty(t, apiResp.Email)
	})

	t.Run("TestVerifyResetToken_AlreadyUsedToken", func(t *testing.T) {
		testUser := setup.TestUsersData["beta_admin"]
		rawToken := getRawResetToken(testUser.Email)

		// Manually mark the token as used in the DB
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2", time.Now(), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually mark token as used")

		verifyPayload := VerifyResetTokenRequest{Token: rawToken}
		verifyBody, err := json.Marshal(verifyPayload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/verify-reset-token", setup.BaseURL), bytes.NewBuffer(verifyBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp VerifyResetTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.Empty(t, apiResp.Email)
	})

	t.Run("TestVerifyResetToken_EmptyToken", func(t *testing.T) {
		verifyPayload := VerifyResetTokenRequest{Token: ""}
		verifyBody, err := json.Marshal(verifyPayload)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/verify-reset-token", setup.BaseURL), bytes.NewBuffer(verifyBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var apiResp VerifyResetTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.False(t, apiResp.Success)
		assert.Empty(t, apiResp.Email)
	})
}

func TestResetPassword(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	ctx := context.Background()

	getRawResetTokenForTest := func(t *testing.T, userEmail string) string {
		t.Helper()
		payload := ForgotPasswordRequest{Email: userEmail}
		body, err := json.Marshal(payload)
		assert.NoError(t, err, "Failed to marshal forgot password payload")

		fpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
		assert.NoError(t, err, "Failed to create forgot password request")
		fpReq.Header.Set("Content-Type", "application/json")

		fpResp, err := client.Do(fpReq)
		assert.NoError(t, err, "Failed to execute forgot password request")
		defer func() {
			if err := fpResp.Body.Close(); err != nil {
				t.Logf("Error closing fpResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, fpResp.StatusCode, "Forgot password request did not return OK")

		emailContent, err := getLatestEmailFromSendgridMock(t, userEmail)
		assert.NoError(t, err, "Failed to get latest email from Sendgrid mock")

		rawToken, err := extractResetTokenFromEmail(t, emailContent)
		assert.NoError(t, err, "Failed to extract reset token from email")
		assert.NotEmpty(t, rawToken, "Extracted reset token should not be empty")
		return rawToken
	}

	t.Run("_ValidTokenAndMatchingPasswords_Successful", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_admin"]
		originalPassword := testUser.PlainTextPassword
		fmt.Println("test user: ", testUser.Email, "original password: ", originalPassword)
		newPassword := "newSecurePassword123"
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resetResp.StatusCode, "Reset password should succeed")
		var apiResp ResetPasswordResponse
		err = json.NewDecoder(resetResp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.True(t, apiResp.Success, "API success should be true for successful password reset")

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, oldLoginResp.StatusCode, "Login with old password should fail after reset")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, newLoginResp.StatusCode, "Login with new password should succeed after reset")

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist in DB after successful reset")
		assert.True(t, dbUsedAt.Valid, "Token should be marked as used after successful reset")
	})

	t.Run("_InvalidToken_NonExistent", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		originalPassword := testUser.PlainTextPassword
		newPassword := "attemptedNewPass1"

		resetPayload := ResetPasswordRequest{
			Token:           "this-token-does-not-exist-12345",
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with non-existent token should fail")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode, "Login with new (attempted) password should fail")

		fmt.Println("Original password for user: ", testUser.Email, "is", originalPassword)
		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode, "Login with old password should still succeed")
	})

	t.Run("_ExpiredToken", func(t *testing.T) {
		testUser := setup.TestUsersData["beta_admin"]
		originalPassword := testUser.PlainTextPassword
		newPassword := "attemptedNewPass2"
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET expires_at = $1 WHERE token_hash = $2", time.Now().Add(-1*time.Hour), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually expire token")

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with expired token should fail")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)
	})

	t.Run("_AlreadyUsedToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		originalPassword := testUser.PlainTextPassword
		newPassword := "attemptedNewPass3"
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2", time.Now(), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually mark token as used")

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with used token should fail")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)
	})

	t.Run("_EmptyToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		originalPassword := testUser.PlainTextPassword
		newPassword := "attemptedNewPass4"

		resetPayload := ResetPasswordRequest{
			Token:           "",
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with empty token should fail")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)
	})

	t.Run("_MissingPassword", func(t *testing.T) {
		testUser := setup.TestUsersData["beta_admin"]
		originalPassword := testUser.PlainTextPassword
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        "",
			PasswordConfirm: "",
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with missing password should fail")

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist after failed reset due to missing password")
		assert.False(t, dbUsedAt.Valid, "Token should NOT be marked as used after failed reset due to missing password")
	})

	t.Run("_PasswordTooShort", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		originalPassword := testUser.PlainTextPassword
		newPassword := "short"
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        newPassword,
			PasswordConfirm: newPassword,
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with short password should fail")

		newLoginResp := attemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist after failed reset due to short password")
		assert.False(t, dbUsedAt.Valid, "Token should NOT be marked as used after failed reset due to short password")
	})

	t.Run("_PasswordsDoNotMatch", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		originalPassword := testUser.PlainTextPassword
		rawToken := getRawResetTokenForTest(t, testUser.Email)

		resetPayload := ResetPasswordRequest{
			Token:           rawToken,
			Password:        "newValidPass123",
			PasswordConfirm: "anotherValidPass456",
		}
		resetBody, err := json.Marshal(resetPayload)
		assert.NoError(t, err)
		resetReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(resetBody))
		assert.NoError(t, err)
		resetReq.Header.Set("Content-Type", "application/json")
		resetResp, err := client.Do(resetReq)
		assert.NoError(t, err)
		defer func() {
			if err := resetResp.Body.Close(); err != nil {
				t.Logf("Error closing resetResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusBadRequest, resetResp.StatusCode, "Reset password with mismatching passwords should fail")

		newLoginResp1 := attemptLogin(t, testUser.Email, "newValidPass123")
		defer func() {
			if err := newLoginResp1.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp1 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp1.StatusCode)

		newLoginResp2 := attemptLogin(t, testUser.Email, "anotherValidPass456")
		defer func() {
			if err := newLoginResp2.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp2 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp2.StatusCode)

		oldLoginResp := attemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusOK, oldLoginResp.StatusCode)

		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist after failed reset due to mismatching passwords")
		assert.False(t, dbUsedAt.Valid, "Token should NOT be marked as used after failed reset due to mismatching passwords")
	})
}
