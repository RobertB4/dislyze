package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangeEmail_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	type ChangeEmailRequest struct {
		NewEmail string `json:"new_email"`
	}

	tests := []struct {
		name             string
		loginUserKey     string
		requestBody      ChangeEmailRequest
		expectedStatus   int
		expectedError    string
		setupFunc        func(t *testing.T)
		validateResponse func(t *testing.T, resp *http.Response)
	}{
		{
			name:         "successful email change request",
			loginUserKey: "enterprise_1",
			requestBody: ChangeEmailRequest{
				NewEmail: "newemail@example.com",
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
				// Verify token was created in database
				ctx := context.Background()
				var tokenCount int
				err := pool.QueryRow(ctx,
					"SELECT COUNT(*) FROM email_change_tokens WHERE user_id = $1 AND new_email = $2",
					setup.TestUsersData2["enterprise_1"].UserID, "newemail@example.com").Scan(&tokenCount)
				assert.NoError(t, err)
				assert.Equal(t, 1, tokenCount, "Email change token should be created")

				// Verify user's current email hasn't changed
				var currentEmail string
				err = pool.QueryRow(ctx,
					"SELECT email FROM users WHERE id = $1",
					setup.TestUsersData2["enterprise_1"].UserID).Scan(&currentEmail)
				assert.NoError(t, err)
				assert.Equal(t, setup.TestUsersData2["enterprise_1"].Email, currentEmail, "User's email should not change yet")
			},
		},
		{
			name:         "empty email",
			loginUserKey: "enterprise_2",
			requestBody: ChangeEmailRequest{
				NewEmail: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "whitespace only email",
			loginUserKey: "enterprise_2",
			requestBody: ChangeEmailRequest{
				NewEmail: "   ",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "email without @ symbol",
			loginUserKey: "enterprise_2",
			requestBody: ChangeEmailRequest{
				NewEmail: "invalidemail.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "email already in use by another user",
			loginUserKey: "enterprise_3",
			requestBody: ChangeEmailRequest{
				NewEmail: setup.TestUsersData2["enterprise_4"].Email, // Use enterprise_4's email
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "このメールアドレスは既に使用されています。",
		},
		{
			name:         "email already in use by user in different tenant",
			loginUserKey: "enterprise_1",
			requestBody: ChangeEmailRequest{
				NewEmail: setup.TestUsersData2["smb_1"].Email, // SMB tenant user's email
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "このメールアドレスは既に使用されています。",
		},
		{
			name:         "user tries to change to their own current email",
			loginUserKey: "enterprise_5",
			requestBody: ChangeEmailRequest{
				NewEmail: setup.TestUsersData2["enterprise_5"].Email,
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "このメールアドレスは既に使用されています。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			client := &http.Client{}
			testUser := setup.TestUsersData2[tt.loginUserKey]
			accessToken, refreshToken := setup.LoginUserAndGetTokens(t, testUser.Email, testUser.PlainTextPassword)

			jsonBody, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-email", setup.BaseURL), bytes.NewBuffer(jsonBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
			req.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedError != "" {
				var errorResp map[string]string
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, errorResp["error"])
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, resp)
			}
		})
	}
}

func TestChangeEmailRateLimit_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use a dedicated user for rate limit testing to avoid interfering with other tests
	rateLimitTestUser := setup.TestUsersData2["enterprise_6"]
	accessToken, refreshToken := setup.LoginUserAndGetTokens(t, rateLimitTestUser.Email, rateLimitTestUser.PlainTextPassword)

	client := &http.Client{}

	// Make multiple requests rapidly
	for i := 0; i < 5; i++ {
		reqBody := map[string]string{
			"new_email": fmt.Sprintf("ratelimit%d@example.com", i),
		}
		jsonBody, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-email", setup.BaseURL), bytes.NewBuffer(jsonBody))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
		req.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken})

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		if i == 0 {
			// First request should succeed
			assert.Equal(t, http.StatusOK, resp.StatusCode, "First request should succeed")
		} else {
			// Subsequent requests should be rate limited
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
				fmt.Sprintf("Request %d should be rate limited", i+1))

			var errorResp map[string]string
			err = json.NewDecoder(resp.Body).Decode(&errorResp)
			assert.NoError(t, err)
			assert.Equal(t, "メールアドレス変更の試行回数が上限を超えました。しばらくしてから再度お試しください。",
				errorResp["error"])
		}
	}
}

func TestChangeEmailAuthentication_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	reqBody := map[string]string{
		"new_email": "unauthorized@example.com",
	}
	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		expectedStatus int
	}{
		{
			name: "no authentication cookies",
			setupRequest: func(req *http.Request) {
				// Don't add any cookies
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid access token",
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: "invalid-token"})
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-email", setup.BaseURL),
				bytes.NewBuffer(jsonBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			tt.setupRequest(req)

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
