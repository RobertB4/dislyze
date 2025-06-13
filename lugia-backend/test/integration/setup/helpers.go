package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/auth"
	"lugia/lib/sendgridlib"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

const (
	BaseURL = "http://lugia-backend:13001/api"
)

func InitDB(t *testing.T) *pgxpool.Pool {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"test_user",
		"test_password",
		"postgres",
		"5432",
		"test_db",
	)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return pool
}

func CleanupDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	t.Log("CleanupDB: Starting database cleanup")

	deleteSQL, err := os.ReadFile("/database/delete.sql")
	if err != nil {
		t.Fatalf("CleanupDB: Failed to read delete.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(deleteSQL))
	if err != nil {
		t.Fatalf("CleanupDB: Failed to execute delete.sql: %v", err)
	}

	t.Log("CleanupDB: Database cleanup completed successfully")
}

func CloseDB(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

func seedDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	t.Log("seedDB: Starting database seeding")

	seedSQL, err := os.ReadFile("/database/seed_localhost.sql")
	if err != nil {
		t.Fatalf("seedDB: Failed to read seed_localhost.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(seedSQL))
	if err != nil {
		t.Fatalf("seedDB: Failed to execute seed_localhost.sql: %v", err)
	}

	t.Log("seedDB: Database seeding completed successfully")
}

func ResetAndSeedDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	CleanupDB(t, pool)
	seedDB(t, pool)
}

func LoginUserAndGetTokens(t *testing.T, email string, password string) (string, string) {
	t.Helper()

	loginPayload := auth.LoginRequestBody{
		Email:    email,
		Password: password,
	}

	payloadBytes, err := json.Marshal(loginPayload)
	assert.NoError(t, err, "Failed to marshal login request")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", BaseURL), bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err, "Failed to create login request")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	assert.NoError(t, err, "Failed to execute login request")
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("Failed to close response body: %v", closeErr)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Login request failed for user %s. Status: %s", email, resp.Status)

	var accessTokenValue string
	var refreshTokenValue string

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		switch cookie.Name {
		case "dislyze_access_token":
			accessTokenValue = cookie.Value
		case "dislyze_refresh_token":
			refreshTokenValue = cookie.Value
		}
	}

	assert.NotEmpty(t, accessTokenValue, "Access token cookie not found or empty after login for %s", email)
	assert.NotEmpty(t, refreshTokenValue, "Refresh token cookie not found or empty after login for %s", email)

	return accessTokenValue, refreshTokenValue
}

func GetLatestEmailFromSendgridMock(t *testing.T, expectedRecipientEmail string) (*sendgridlib.SendGridMailRequestBody, error) {
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

		var emails []sendgridlib.SendGridMailRequestBody
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

func ExtractResetTokenFromEmail(t *testing.T, email *sendgridlib.SendGridMailRequestBody) (string, error) {
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

func AttemptLogin(t *testing.T, email string, password string) *http.Response {
	t.Helper()
	client := &http.Client{}

	loginPayload := auth.LoginRequestBody{
		Email:    email,
		Password: password,
	}
	loginBody, err := json.Marshal(loginPayload)
	assert.NoError(t, err, "Failed to marshal login payload in attemptLogin")

	loginReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", BaseURL), bytes.NewBuffer(loginBody))
	assert.NoError(t, err, "Failed to create login request in attemptLogin")
	loginReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(loginReq)
	assert.NoError(t, err, "Failed to execute login request in attemptLogin")
	return resp
}
