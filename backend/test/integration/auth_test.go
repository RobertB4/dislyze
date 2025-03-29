package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	baseURL = "http://backend:1337"
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

func TestSignup(t *testing.T) {
	// Clean up database before tests
	CleanupDB(t)
	defer CloseDB()

	tests := []struct {
		name           string
		request        SignupRequest
		expectedStatus int
		expectedError  string
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
			expectedError:  "company name is required",
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
			expectedError:  "user name is required",
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
			expectedError:  "email is required",
		},
		{
			name: "missing password",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password is required",
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
			expectedError:  "password must be at least 8 characters long",
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
			expectedError:  "passwords do not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			// Create request
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", baseURL), bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var response SignupResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			// Check response
			if tt.expectedError != "" {
				assert.False(t, response.Success)
				assert.Equal(t, tt.expectedError, response.Error)
			} else {
				assert.True(t, response.Success)
				assert.Empty(t, response.Error)

				// Check cookies
				cookies := resp.Cookies()
				assert.NotEmpty(t, cookies, "Expected cookies in response")

				var accessToken, refreshToken *http.Cookie
				for _, cookie := range cookies {
					switch cookie.Name {
					case "access_token":
						accessToken = cookie
					case "refresh_token":
						refreshToken = cookie
					}
				}

				// Verify access token cookie
				assert.NotNil(t, accessToken, "Access token cookie not found")
				assert.True(t, accessToken.HttpOnly, "Access token cookie should be HttpOnly")
				assert.True(t, accessToken.Secure, "Access token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, accessToken.SameSite, "Access token cookie should have SameSite=Strict")

				// Verify refresh token cookie
				assert.NotNil(t, refreshToken, "Refresh token cookie not found")
				assert.True(t, refreshToken.HttpOnly, "Refresh token cookie should be HttpOnly")
				assert.True(t, refreshToken.Secure, "Refresh token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, refreshToken.SameSite, "Refresh token cookie should have SameSite=Strict")
			}
		})
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	// Clean up database before tests
	CleanupDB(t)
	defer CloseDB()

	// First signup
	request := SignupRequest{
		CompanyName:     "Test Company",
		UserName:        "Test User",
		Email:           "duplicate@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}

	body, err := json.Marshal(request)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", baseURL), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Second signup with same email

	req2, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/signup", baseURL), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var response SignupResponse
	err = json.NewDecoder(resp2.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "user with this email already exists", response.Error)
}
