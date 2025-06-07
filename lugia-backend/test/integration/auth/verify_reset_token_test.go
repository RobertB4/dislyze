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

func TestVerifyResetToken(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	ctx := context.Background()

	// Helper function to make a /auth/forgot-password request and get the raw token
	getRawResetToken := func(userEmail string) string {
		payload := auth.ForgotPasswordRequestBody{Email: userEmail}
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

		email, err := setup.GetLatestEmailFromSendgridMock(t, userEmail)
		assert.NoError(t, err)
		rawToken, err := setup.ExtractResetTokenFromEmail(t, email)
		assert.NoError(t, err)
		assert.NotEmpty(t, rawToken)
		return rawToken
	}

	t.Run("TestVerifyResetToken_ValidToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_admin"]
		rawToken := getRawResetToken(testUser.Email)
		fmt.Println("Raw token for user:", testUser.Email, "is", rawToken)

		verifyPayload := auth.VerifyResetTokenRequestBody{Token: rawToken}
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
		var apiResp map[string]string
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		assert.NoError(t, err)
		assert.Equal(t, testUser.Email, apiResp["email"])

		// Verify DB token is not marked as used
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		var dbUsedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT used_at FROM password_reset_tokens WHERE token_hash = $1", hashedTokenStr).Scan(&dbUsedAt)
		assert.NoError(t, err, "Token should still exist")
		assert.False(t, dbUsedAt.Valid, "Token should not be marked as used after verification")
	})

	t.Run("TestVerifyResetToken_InvalidToken_NonExistent", func(t *testing.T) {
		verifyPayload := auth.VerifyResetTokenRequestBody{Token: "non-existent-token-string"}
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
	})

	t.Run("TestVerifyResetToken_ExpiredToken", func(t *testing.T) {
		testUser := setup.TestUsersData["alpha_editor"]
		rawToken := getRawResetToken(testUser.Email)

		// Manually expire the token in the DB
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET expires_at = $1 WHERE token_hash = $2", time.Now().Add(-1*time.Hour), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually expire token")

		verifyPayload := auth.VerifyResetTokenRequestBody{Token: rawToken}
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
	})

	t.Run("TestVerifyResetToken_AlreadyUsedToken", func(t *testing.T) {
		testUser := setup.TestUsersData["beta_admin"]
		rawToken := getRawResetToken(testUser.Email)

		// Manually mark the token as used in the DB
		hash := sha256.Sum256([]byte(rawToken))
		hashedTokenStr := hex.EncodeToString(hash[:])
		_, err := pool.Exec(ctx, "UPDATE password_reset_tokens SET used_at = $1 WHERE token_hash = $2", time.Now(), hashedTokenStr)
		assert.NoError(t, err, "Failed to manually mark token as used")

		verifyPayload := auth.VerifyResetTokenRequestBody{Token: rawToken}
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
	})

	t.Run("TestVerifyResetToken_EmptyToken", func(t *testing.T) {
		verifyPayload := auth.VerifyResetTokenRequestBody{Token: ""}
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
	})
}
