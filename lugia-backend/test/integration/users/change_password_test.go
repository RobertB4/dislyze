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

func TestChangePassword_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	type ChangePasswordRequest struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		PasswordConfirm string `json:"new_password_confirm"`
	}

	tests := []struct {
		name           string
		loginUserKey   string
		requestBody    ChangePasswordRequest
		expectedStatus int
		expectUnauth   bool
	}{
		{
			name:         "invalid current password",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "wrongCurrentPassword",
				NewPassword:     "newSecurePassword123",
				PasswordConfirm: "newSecurePassword123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "password mismatch",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: setup.TestUsersData2["enterprise_1"].PlainTextPassword,
				NewPassword:     "newSecurePassword123",
				PasswordConfirm: "differentPassword123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "new password too short",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: setup.TestUsersData2["enterprise_1"].PlainTextPassword,
				NewPassword:     "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "empty current password",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "",
				NewPassword:     "newSecurePassword123",
				PasswordConfirm: "newSecurePassword123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "empty new password",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: setup.TestUsersData2["enterprise_1"].PlainTextPassword,
				NewPassword:     "",
				PasswordConfirm: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unauthenticated request",
			requestBody: ChangePasswordRequest{
				CurrentPassword: "somePassword",
				NewPassword:     "newSecurePassword123",
				PasswordConfirm: "newSecurePassword123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:         "successful password change",
			loginUserKey: "enterprise_1",
			requestBody: ChangePasswordRequest{
				CurrentPassword: setup.TestUsersData2["enterprise_1"].PlainTextPassword,
				NewPassword:     "newSecurePassword123",
				PasswordConfirm: "newSecurePassword123",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}

			payloadBytes, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err, "Failed to marshal request body for test: %s", tt.name)

			reqURL := fmt.Sprintf("%s/me/change-password", setup.BaseURL)
			req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(payloadBytes))
			assert.NoError(t, err, "Failed to create request for test: %s", tt.name)
			req.Header.Set("Content-Type", "application/json")

			if !tt.expectUnauth && tt.loginUserKey != "" {
				userDetails, ok := setup.TestUsersData2[tt.loginUserKey]
				assert.True(t, ok, "User key '%s' not found in setup.TestUsersData", tt.loginUserKey)

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, userDetails.Email, userDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
				req.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request for test: %s", tt.name)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s", tt.name)

			if tt.expectedStatus == http.StatusOK {
				// Test that the new password works for login
				attemptLoginResp := setup.AttemptLogin(t, setup.TestUsersData2[tt.loginUserKey].Email, tt.requestBody.NewPassword)
				defer func() {
					if err := attemptLoginResp.Body.Close(); err != nil {
						t.Logf("Error closing login attempt response body: %v", err)
					}
				}()
				assert.Equal(t, http.StatusOK, attemptLoginResp.StatusCode, "Login with new password should succeed for test: %s", tt.name)

				// Test that the old password no longer works
				attemptOldLoginResp := setup.AttemptLogin(t, setup.TestUsersData2[tt.loginUserKey].Email, setup.TestUsersData2[tt.loginUserKey].PlainTextPassword)
				defer func() {
					if err := attemptOldLoginResp.Body.Close(); err != nil {
						t.Logf("Error closing old login attempt response body: %v", err)
					}
				}()
				assert.Equal(t, http.StatusUnauthorized, attemptOldLoginResp.StatusCode, "Login with old password should fail for test: %s", tt.name)

			} else if tt.expectedStatus == http.StatusBadRequest || tt.expectedStatus == http.StatusUnauthorized {
				// Only "invalid current password" case should have an error message in the response body
				if tt.name == "invalid current password" {
					var errorResponse ErrorResponse
					err = json.NewDecoder(resp.Body).Decode(&errorResponse)
					assert.NoError(t, err, "Failed to decode error response for test: %s", tt.name)
					assert.NotEmpty(t, errorResponse.Error, "Expected error message to be present for test: %s", tt.name)
				}
			}
		})
	}
}

func TestChangePasswordSessionInvalidation_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	testUser := setup.TestUsersData2["enterprise_1"]

	// Login to get initial tokens
	accessToken1, refreshToken1 := setup.LoginUserAndGetTokens(t, testUser.Email, testUser.PlainTextPassword)

	// Login again to create another session (to test that ALL refresh tokens are invalidated)
	accessToken2, refreshToken2 := setup.LoginUserAndGetTokens(t, testUser.Email, testUser.PlainTextPassword)

	// Verify both sessions work by calling /me
	meReq1, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)
	meReq1.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken1})
	meReq1.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken1})

	meResp1, err := client.Do(meReq1)
	assert.NoError(t, err)
	defer func() {
		if err := meResp1.Body.Close(); err != nil {
			t.Logf("Error closing meResp1 body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, meResp1.StatusCode, "First session should work before password change")

	meReq2, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)
	meReq2.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken2})
	meReq2.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken2})

	meResp2, err := client.Do(meReq2)
	assert.NoError(t, err)
	defer func() {
		if err := meResp2.Body.Close(); err != nil {
			t.Logf("Error closing meResp2 body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, meResp2.StatusCode, "Second session should work before password change")

	// Change password using the first session
	changePasswordReq := struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		PasswordConfirm string `json:"new_password_confirm"`
	}{
		CurrentPassword: testUser.PlainTextPassword,
		NewPassword:     "newSecurePassword123",
		PasswordConfirm: "newSecurePassword123",
	}

	payloadBytes, err := json.Marshal(changePasswordReq)
	assert.NoError(t, err)

	cpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-password", setup.BaseURL), bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	cpReq.Header.Set("Content-Type", "application/json")
	cpReq.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken1})
	cpReq.AddCookie(&http.Cookie{Name: "dislyze_refresh_token", Value: refreshToken1})

	cpResp, err := client.Do(cpReq)
	assert.NoError(t, err)
	defer func() {
		if err := cpResp.Body.Close(); err != nil {
			t.Logf("Error closing cpResp body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, cpResp.StatusCode, "Password change should succeed")

	// Try to use the first session again - should still work since access token is still valid
	// but refresh token should be invalidated
	meReq1After, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)
	meReq1After.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken1})

	meResp1After, err := client.Do(meReq1After)
	assert.NoError(t, err)
	defer func() {
		if err := meResp1After.Body.Close(); err != nil {
			t.Logf("Error closing meResp1After body: %v", err)
		}
	}()
	// Access token should still work until it expires naturally
	assert.Equal(t, http.StatusOK, meResp1After.StatusCode, "Access token should still work after password change")

	// Try to use the second session - access token should still work
	meReq2After, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	assert.NoError(t, err)
	meReq2After.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken2})

	meResp2After, err := client.Do(meReq2After)
	assert.NoError(t, err)
	defer func() {
		if err := meResp2After.Body.Close(); err != nil {
			t.Logf("Error closing meResp2After body: %v", err)
		}
	}()
	// Access token should still work until it expires naturally
	assert.Equal(t, http.StatusOK, meResp2After.StatusCode, "Second session access token should still work after password change")

	// Verify that refresh tokens are invalidated by checking the database
	ctx := context.Background()
	var refreshTokenCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1", testUser.UserID).Scan(&refreshTokenCount)
	assert.NoError(t, err, "Failed to query refresh token count")
	assert.Equal(t, 0, refreshTokenCount, "All refresh tokens should be deleted after password change")
}
