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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestForgotPassword(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	t.Run("TestForgotPassword_ExistingEmail_Successful", func(t *testing.T) {
		testUser := setup.TestUsersData["enterprise_1"]
		payload := auth.ForgotPasswordRequestBody{Email: testUser.Email}
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

		email, err := setup.GetLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err, "Failed to get email from SendGrid mock")
		if err == nil {
			assert.Equal(t, "パスワードリセットのご案内 - dislyze", email.Personalizations[0].Subject)

			rawToken, err := setup.ExtractResetTokenFromEmail(t, email)
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
		payload := auth.ForgotPasswordRequestBody{Email: nonExistentEmail}
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
	})

	t.Run("TestForgotPassword_InvalidEmailFormat", func(t *testing.T) {
		payload := auth.ForgotPasswordRequestBody{Email: "invalidemail"}
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
	})

	t.Run("TestForgotPassword_EmptyEmail", func(t *testing.T) {
		payload := auth.ForgotPasswordRequestBody{Email: ""}
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
	})

	t.Run("TestForgotPassword_MultipleRequestsForSameUser", func(t *testing.T) {
		testUser := setup.TestUsersData["enterprise_2"]

		// --- First Request ---
		payload1 := auth.ForgotPasswordRequestBody{Email: testUser.Email}
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

		email1, err := setup.GetLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err)
		rawToken1, err := setup.ExtractResetTokenFromEmail(t, email1)
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
		payload2 := auth.ForgotPasswordRequestBody{Email: testUser.Email}
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

		email2, err := setup.GetLatestEmailFromSendgridMock(t, testUser.Email)
		assert.NoError(t, err)
		rawToken2, err := setup.ExtractResetTokenFromEmail(t, email2)
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
