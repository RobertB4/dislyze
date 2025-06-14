package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"giratina/features/auth"
	"giratina/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginBasic(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Test with internal admin user (should succeed in giratina)
	testUser := setup.TestUsersData["internal_1"]

	loginPayload := auth.LoginRequestBody{
		Email:    testUser.Email,
		Password: testUser.PlainTextPassword,
	}

	body, err := json.Marshal(loginPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/login", setup.BaseURL), bytes.NewBuffer(body))
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

	// Should succeed for internal admin user
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify cookies are set
	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies, "Expected cookies in response for successful login")

	var accessToken, refreshToken *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "dislyze_access_token":
			accessToken = cookie
		case "dislyze_refresh_token":
			refreshToken = cookie
		}
	}

	assert.NotNil(t, accessToken, "Access token cookie not found")
	assert.NotNil(t, refreshToken, "Refresh token cookie not found")
}