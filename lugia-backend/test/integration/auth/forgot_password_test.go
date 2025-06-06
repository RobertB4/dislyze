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
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func TestForgotPasswordValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		request        ForgotPasswordRequest
		expectedStatus int
	}{
		{
			name: "missing email",
			request: ForgotPasswordRequest{
				Email: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email format",
			request: ForgotPasswordRequest{
				Email: "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "valid email (non-existent user)",
			request: ForgotPasswordRequest{
				Email: "nonexistent@example.com",
			},
			expectedStatus: http.StatusOK, // Always returns 200 for security
		},
		{
			name: "valid email format",
			request: ForgotPasswordRequest{
				Email: "test@example.com",
			},
			expectedStatus: http.StatusOK, // Always returns 200 for security
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/forgot-password", setup.BaseURL), bytes.NewBuffer(body))
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

			// For forgot password, no cookies should ever be set
			cookies := resp.Cookies()
			assert.Empty(t, cookies, "Expected no cookies for forgot password")
		})
	}
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

func TestForgotPasswordComplex(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
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