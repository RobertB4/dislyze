package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogout(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	createTestUser(t)

	client := &http.Client{}

	// First, log in to get cookies
	loginPayload := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginBody, err := json.Marshal(loginPayload)
	assert.NoError(t, err)

	loginReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(loginBody))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer func() {
		if err := loginResp.Body.Close(); err != nil {
			t.Logf("Error closing loginResp body: %v", err)
		}
	}()
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	loginCookies := loginResp.Cookies()
	assert.NotEmpty(t, loginCookies, "Expected cookies from login")

	// Now test logout
	logoutReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/logout", setup.BaseURL), nil)
	assert.NoError(t, err)
	for _, cookie := range loginCookies {
		logoutReq.AddCookie(cookie)
	}

	logoutResp, err := client.Do(logoutReq)
	assert.NoError(t, err)
	defer func() {
		if err := logoutResp.Body.Close(); err != nil {
			t.Logf("Error closing logoutResp body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, logoutResp.StatusCode, "Logout should return 200 OK")

	// Verify that the logout response clears the cookies
	var accessTokenCleared, refreshTokenCleared bool
	for _, cookie := range logoutResp.Cookies() {
		if cookie.Name == "dislyze_access_token" {
			assert.Equal(t, "", cookie.Value, "Access token should be cleared")
			assert.Equal(t, -1, cookie.MaxAge, "Access token MaxAge should be -1")
			accessTokenCleared = true
		}
		if cookie.Name == "dislyze_refresh_token" {
			assert.Equal(t, "", cookie.Value, "Refresh token should be cleared")
			assert.Equal(t, -1, cookie.MaxAge, "Refresh token MaxAge should be -1")
			refreshTokenCleared = true
		}
	}
	assert.True(t, accessTokenCleared, "Access token clear instruction not found")
	assert.True(t, refreshTokenCleared, "Refresh token clear instruction not found")
}

func TestLogoutWithoutCookies(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	// Test logout without being logged in (no cookies)
	logoutReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/logout", setup.BaseURL), nil)
	assert.NoError(t, err)

	logoutResp, err := client.Do(logoutReq)
	assert.NoError(t, err)
	defer func() {
		if err := logoutResp.Body.Close(); err != nil {
			t.Logf("Error closing logoutResp body: %v", err)
		}
	}()

	// Logout should still work even without cookies (idempotent)
	assert.Equal(t, http.StatusOK, logoutResp.StatusCode, "Logout should return 200 OK even without cookies")

	// Should still send cookie clearing instructions
	var accessTokenCleared, refreshTokenCleared bool
	for _, cookie := range logoutResp.Cookies() {
		if cookie.Name == "dislyze_access_token" {
			assert.Equal(t, "", cookie.Value, "Access token should be cleared")
			assert.Equal(t, -1, cookie.MaxAge, "Access token MaxAge should be -1")
			accessTokenCleared = true
		}
		if cookie.Name == "dislyze_refresh_token" {
			assert.Equal(t, "", cookie.Value, "Refresh token should be cleared")
			assert.Equal(t, -1, cookie.MaxAge, "Refresh token MaxAge should be -1")
			refreshTokenCleared = true
		}
	}
	assert.True(t, accessTokenCleared, "Access token clear instruction not found")
	assert.True(t, refreshTokenCleared, "Refresh token clear instruction not found")
}