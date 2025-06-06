package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

type ResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func TestResetPasswordValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		request        ResetPasswordRequest
		expectedStatus int
	}{
		{
			name: "missing token",
			request: ResetPasswordRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: ResetPasswordRequest{
				Token:           "some-token",
				Password:        "",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			request: ResetPasswordRequest{
				Token:           "some-token",
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "passwords do not match",
			request: ResetPasswordRequest{
				Token:           "some-token",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid token (valid format but doesn't exist)",
			request: ResetPasswordRequest{
				Token:           "invalid-token-123",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest, // Token not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/reset-password", setup.BaseURL), bytes.NewBuffer(body))
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

			// For reset password, no cookies should ever be set
			cookies := resp.Cookies()
			assert.Empty(t, cookies, "Expected no cookies for reset password")
		})
	}
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

func TestResetPasswordComplex(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
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
}