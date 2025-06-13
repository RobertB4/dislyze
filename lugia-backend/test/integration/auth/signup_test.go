package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SignupRequestBody struct {
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func TestSignup(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		request        SignupRequestBody
		expectedStatus int
	}{
		{
			name: "successful signup",
			request: SignupRequestBody{
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
			request: SignupRequestBody{
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing user name",
			request: SignupRequestBody{
				CompanyName:     "Test Company",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing email",
			request: SignupRequestBody{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: SignupRequestBody{
				CompanyName: "Test Company",
				UserName:    "Test User",
				Email:       "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			request: SignupRequestBody{
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
			request: SignupRequestBody{
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
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	request := SignupRequestBody{
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

	var errorResponse map[string]any
	err = json.NewDecoder(resp2.Body).Decode(&errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "このメールアドレスは既に使用されています。", errorResponse["error"])

	// Expect no cookies for the failed duplicate email signup
	cookies2 := resp2.Cookies()
	assert.Empty(t, cookies2, "Expected no cookies for duplicate email signup attempt")
}
