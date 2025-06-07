package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lugia/features/auth"
	"lugia/test/integration/setup"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestResetPassword(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	ctx := context.Background()

	getRawResetTokenForTest := func(t *testing.T, userEmail string) string {
		t.Helper()
		payload := auth.ForgotPasswordRequestBody{Email: userEmail}
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

		emailContent, err := setup.GetLatestEmailFromSendgridMock(t, userEmail)
		assert.NoError(t, err, "Failed to get latest email from Sendgrid mock")

		rawToken, err := setup.ExtractResetTokenFromEmail(t, emailContent)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
		defer func() {
			if err := oldLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing oldLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, oldLoginResp.StatusCode, "Login with old password should fail after reset")

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode, "Login with new (attempted) password should fail")

		fmt.Println("Original password for user: ", testUser.Email, "is", originalPassword)
		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp := setup.AttemptLogin(t, testUser.Email, newPassword)
		defer func() {
			if err := newLoginResp.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp.StatusCode)

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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

		resetPayload := auth.ResetPasswordRequestBody{
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

		newLoginResp1 := setup.AttemptLogin(t, testUser.Email, "newValidPass123")
		defer func() {
			if err := newLoginResp1.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp1 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp1.StatusCode)

		newLoginResp2 := setup.AttemptLogin(t, testUser.Email, "anotherValidPass456")
		defer func() {
			if err := newLoginResp2.Body.Close(); err != nil {
				t.Logf("Error closing newLoginResp2 body: %v", err)
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, newLoginResp2.StatusCode)

		oldLoginResp := setup.AttemptLogin(t, testUser.Email, originalPassword)
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
