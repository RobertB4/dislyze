package users

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestVerifyChangeEmail_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Helper function to create deterministic test tokens
	createTestToken := func(seed byte) (plaintext, hash string) {
		tokenBytes := make([]byte, 32)
		for i := range tokenBytes {
			tokenBytes[i] = seed // Use seed for deterministic tokens
		}
		plaintext = base64.URLEncoding.EncodeToString(tokenBytes)
		hashBytes := sha256.Sum256([]byte(plaintext))
		hash = fmt.Sprintf("%x", hashBytes[:])
		return
	}

	ctx := context.Background()
	client := &http.Client{}

	// Create test tokens for various scenarios
	expiredPlaintext, expiredHash := createTestToken(1)
	invalidPlaintext, _ := createTestToken(2)

	// Insert expired token directly into database (expired 1 hour ago)
	validUser := setup.TestUsersData["enterprise_1"]
	expiredTime := time.Now().Add(-1 * time.Hour)
	_, err := pool.Exec(ctx,
		"INSERT INTO email_change_tokens (user_id, new_email, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
		validUser.UserID, "expired@example.com", expiredHash, expiredTime)
	assert.NoError(t, err, "Should insert expired token")

	// Login user to get access token for authenticated requests
	accessToken, _ := setup.LoginUserAndGetTokens(t, validUser.Email, validUser.PlainTextPassword)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れのトークンです。",
		},
		{
			name:           "invalid token",
			token:          invalidPlaintext,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れのトークンです。",
		},
		{
			name:           "expired token",
			token:          expiredPlaintext,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れのトークンです。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.token == "" {
				url = fmt.Sprintf("%s/me/verify-change-email", setup.BaseURL)
			} else {
				url = fmt.Sprintf("%s/me/verify-change-email?token=%s", setup.BaseURL, tt.token)
			}

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			// Add authentication cookie
			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

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
		})
	}

	// Test successful verification
	t.Run("successful verification", func(t *testing.T) {
		testUser := setup.TestUsersData["enterprise_2"]
		successEmail := "success.verify@example.com"
		successPlaintext, successHash := createTestToken(99)

		// Insert known token into database
		expiresAt := time.Now().Add(30 * time.Minute)
		_, err = pool.Exec(ctx,
			"INSERT INTO email_change_tokens (user_id, new_email, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
			testUser.UserID, successEmail, successHash, expiresAt)
		assert.NoError(t, err)

		// Verify with our known token
		verifyURL := fmt.Sprintf("%s/me/verify-change-email?token=%s", setup.BaseURL, successPlaintext)
		verifyReq, err := http.NewRequest("GET", verifyURL, nil)
		assert.NoError(t, err)

		// Add authentication cookie
		verifyReq.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		verifyResp, err := client.Do(verifyReq)
		assert.NoError(t, err)
		defer verifyResp.Body.Close()

		assert.Equal(t, http.StatusOK, verifyResp.StatusCode, "Email verification should succeed")

		// Test that user can now login with new email and old password
		loginReq := map[string]string{
			"email":    successEmail,
			"password": testUser.PlainTextPassword,
		}
		loginBody, err := json.Marshal(loginReq)
		assert.NoError(t, err)

		loginHTTPReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(loginBody))
		assert.NoError(t, err)
		loginHTTPReq.Header.Set("Content-Type", "application/json")

		loginResp, err := client.Do(loginHTTPReq)
		assert.NoError(t, err)
		defer loginResp.Body.Close()

		assert.Equal(t, http.StatusOK, loginResp.StatusCode, "Should be able to login with new email")

		// Verify token was marked as used
		var usedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx,
			"SELECT used_at FROM email_change_tokens WHERE token_hash = $1",
			successHash).Scan(&usedAt)
		assert.NoError(t, err)
		assert.True(t, usedAt.Valid, "Email change token should be marked as used")
		assert.True(t, usedAt.Time.After(time.Now().Add(-1*time.Minute)), "used_at should be recent")

		// Note: We expect 1 refresh token here because the login above creates a new one
		// The VerifyChangeEmail endpoint should have deleted all old tokens, and the login creates a fresh one
		var refreshTokenCount int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1",
			testUser.UserID).Scan(&refreshTokenCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, refreshTokenCount, "Should have 1 refresh token from the login after email change")
	})

	// Test token reuse (should fail on second attempt)
	t.Run("token reuse fails", func(t *testing.T) {
		reuseUser := setup.TestUsersData["smb_1"]
		reuseEmail := "reuse.test@example.com"
		reusePlaintext, reuseHash := createTestToken(88)

		expiresAt := time.Now().Add(30 * time.Minute)
		_, err := pool.Exec(ctx,
			"INSERT INTO email_change_tokens (user_id, new_email, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
			reuseUser.UserID, reuseEmail, reuseHash, expiresAt)
		assert.NoError(t, err)

		verifyURL := fmt.Sprintf("%s/me/verify-change-email?token=%s", setup.BaseURL, reusePlaintext)

		// First verification should succeed
		verifyReq1, err := http.NewRequest("GET", verifyURL, nil)
		assert.NoError(t, err)

		// Add authentication cookie
		verifyReq1.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		verifyResp1, err := client.Do(verifyReq1)
		assert.NoError(t, err)
		defer verifyResp1.Body.Close()

		assert.Equal(t, http.StatusOK, verifyResp1.StatusCode, "First verification should succeed")

		// Second verification should fail (token already used)
		verifyReq2, err := http.NewRequest("GET", verifyURL, nil)
		assert.NoError(t, err)

		// Add authentication cookie
		verifyReq2.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		verifyResp2, err := client.Do(verifyReq2)
		assert.NoError(t, err)
		defer verifyResp2.Body.Close()

		assert.Equal(t, http.StatusBadRequest, verifyResp2.StatusCode, "Second verification should fail")

		var errorResp map[string]string
		err = json.NewDecoder(verifyResp2.Body).Decode(&errorResp)
		assert.NoError(t, err)
		assert.Equal(t, "無効または期限切れのトークンです。", errorResp["error"])
	})

	// Test unauthenticated access (should fail with 401)
	t.Run("unauthenticated access returns 401", func(t *testing.T) {
		// Create a valid token for testing
		unauthPlaintext, unauthHash := createTestToken(77)
		unauthUser := setup.TestUsersData["enterprise_1"]

		// Clean up any existing tokens for this user first
		_, err := pool.Exec(ctx, "DELETE FROM email_change_tokens WHERE user_id = $1", unauthUser.UserID)
		assert.NoError(t, err)

		// Insert token into database
		expiresAt := time.Now().Add(30 * time.Minute)
		_, err = pool.Exec(ctx,
			"INSERT INTO email_change_tokens (user_id, new_email, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
			unauthUser.UserID, "unauth@example.com", unauthHash, expiresAt)
		assert.NoError(t, err)

		// Try to verify without authentication
		verifyURL := fmt.Sprintf("%s/me/verify-change-email?token=%s", setup.BaseURL, unauthPlaintext)
		req, err := http.NewRequest("GET", verifyURL, nil)
		assert.NoError(t, err)

		// NO authentication cookie added

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Unauthenticated request should return 401")
	})
}
