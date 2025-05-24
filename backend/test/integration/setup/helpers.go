package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

const (
	BaseURL = "http://backend:13001"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

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

	deleteSQL, err := os.ReadFile("/database/delete.sql")
	if err != nil {
		t.Fatalf("Failed to read delete.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(deleteSQL))
	if err != nil {
		t.Fatalf("Failed to execute delete.sql: %v", err)
	}
}

func CloseDB(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

func SeedDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	seedSQL, err := os.ReadFile("setup/seed.sql")
	if err != nil {
		t.Fatalf("Failed to read seed.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(seedSQL))
	if err != nil {
		t.Fatalf("Failed to execute seed.sql: %v", err)
	}
}

func LoginUserAndGetTokens(t *testing.T, email string, password string) (string, string) {
	t.Helper()

	loginPayload := LoginRequest{
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
	defer resp.Body.Close()

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
