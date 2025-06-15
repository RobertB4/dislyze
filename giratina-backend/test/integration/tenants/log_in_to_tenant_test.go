package tenants

import (
	"fmt"
	"giratina/test/integration/setup"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogInToTenant_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map (employee who logs in)
		tenantID       string // Tenant ID to log into
		expectedStatus int
		expectCookies  bool
	}{
		// Happy Path Test
		{
			name:           "internal_1 logs into enterprise tenant",
			loginUserKey:   "internal_1",
			tenantID:       "11111111-1111-1111-1111-111111111111", // Enterprise tenant
			expectedStatus: http.StatusOK,
			expectCookies:  true,
		},
		{
			name:           "internal_2 logs into SMB tenant",
			loginUserKey:   "internal_2",
			tenantID:       "22222222-2222-2222-2222-222222222222", // SMB tenant
			expectedStatus: http.StatusOK,
			expectCookies:  true,
		},

		// Authentication Security Tests
		{
			name:           "unauthenticated request returns 401",
			loginUserKey:   "", // No login
			tenantID:       "11111111-1111-1111-1111-111111111111",
			expectedStatus: http.StatusUnauthorized,
			expectCookies:  false,
		},

		// Input Validation Tests
		{
			name:           "invalid UUID format returns 400",
			loginUserKey:   "internal_1",
			tenantID:       "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			expectCookies:  false,
		},
		{
			name:           "non-existent tenant returns 401",
			loginUserKey:   "internal_1",
			tenantID:       "99999999-9999-9999-9999-999999999999", // Doesn't exist
			expectedStatus: http.StatusUnauthorized,
			expectCookies:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie

			// Login as employee if specified
			if tt.loginUserKey != "" {
				currentUserDetails, ok := setup.TestUsersData[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: User key '%s' not found in TestUsersData", tt.loginUserKey)
				}

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			// Make request to tenant login endpoint
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/tenants/%s/login", setup.BaseURL, tt.tenantID), nil)
			assert.NoError(t, err)

			// Add cookies if we have them
			if len(cookies) > 0 {
				for _, cookie := range cookies {
					req.AddCookie(cookie)
				}
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Verify status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectCookies {
				// Verify success response
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Verify cookies are set
				responseCookies := resp.Cookies()
				var accessTokenFound, refreshTokenFound bool
				for _, cookie := range responseCookies {
					switch cookie.Name {
					case "dislyze_access_token":
						accessTokenFound = true
						assert.NotEmpty(t, cookie.Value, "Access token should not be empty")
						assert.Equal(t, "/", cookie.Path, "Access token path should be /")
						assert.True(t, cookie.HttpOnly, "Access token should be HttpOnly")
					case "dislyze_refresh_token":
						refreshTokenFound = true
						assert.NotEmpty(t, cookie.Value, "Refresh token should not be empty")
						assert.Equal(t, "/", cookie.Path, "Refresh token path should be /")
						assert.True(t, cookie.HttpOnly, "Refresh token should be HttpOnly")
						assert.Equal(t, 7*24*60*60, cookie.MaxAge, "Refresh token should expire in 7 days")
					}
				}
				assert.True(t, accessTokenFound, "Access token cookie should be set")
				assert.True(t, refreshTokenFound, "Refresh token cookie should be set")

			} else {
				// Verify error response
				assert.NotEqual(t, http.StatusOK, resp.StatusCode)

				// Read error response for debugging
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Error response for %s: %s", tt.name, string(bodyBytes))

				// Verify no cookies are set
				responseCookies := resp.Cookies()
				for _, cookie := range responseCookies {
					if cookie.Name == "dislyze_access_token" || cookie.Name == "dislyze_refresh_token" {
						t.Errorf("No auth cookies should be set on error, but found: %s", cookie.Name)
					}
				}
			}
		})
	}
}

func TestLogInToTenant_EmptyTenantID_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	// Login as employee
	currentUserDetails := setup.TestUsersData["internal_1"]
	accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
	cookies := []*http.Cookie{
		{Name: "dislyze_access_token", Value: accessToken},
		{Name: "dislyze_refresh_token", Value: refreshToken},
	}

	// Make request with empty tenant ID (this should hit the route parameter validation)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tenants//login", setup.BaseURL), nil)
	assert.NoError(t, err)

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	// Should return 400 because empty tenant ID is invalid
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLogInToTenant_InvalidJWT_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name        string
		accessToken string
		tenantID    string
	}{
		{
			name:        "malformed JWT token",
			accessToken: "invalid.jwt.token",
			tenantID:    "11111111-1111-1111-1111-111111111111",
		},
		{
			name:        "empty JWT token",
			accessToken: "",
			tenantID:    "11111111-1111-1111-1111-111111111111",
		},
		{
			name:        "expired JWT token",
			accessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDk0NTkyMDB9.invalid", // Expired token
			tenantID:    "11111111-1111-1111-1111-111111111111",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/tenants/%s/login", setup.BaseURL, tt.tenantID), nil)
			assert.NoError(t, err)

			if tt.accessToken != "" {
				req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: tt.accessToken})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}