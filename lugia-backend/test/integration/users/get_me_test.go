package users

import (
	"encoding/json"
	"fmt"
	"io"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMe_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                   string
		loginUserKey           string // Key for setup.TestUsersData map
		expectedStatus         int
		expectedTenantName     string
		expectedMinPermissions int // Minimum number of permissions expected
		expectErrorResponse    bool
	}{
		{
			name:                   "enterprise_1 gets their details",
			loginUserKey:           "enterprise_1",
			expectedStatus:         http.StatusOK,
			expectedTenantName:     "エンタープライズ株式会社",
			expectedMinPermissions: 3,
		},
		{
			name:                   "enterprise_2 gets their details",
			loginUserKey:           "enterprise_2",
			expectedStatus:         http.StatusOK,
			expectedTenantName:     "エンタープライズ株式会社",
			expectedMinPermissions: 0, // Should have exactly 0 permissions
		},
		{
			name:                "unauthenticated user gets 401",
			loginUserKey:        "", // No login
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var cookies []*http.Cookie
			var currentUserDetails setup.UserTestData

			if tt.loginUserKey != "" {
				var ok bool
				currentUserDetails, ok = setup.TestUsersData2[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: User key '%s' not found in TestUsersData", tt.loginUserKey)
				}

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			req, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
			assert.NoError(t, err)

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

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var meResponse users.MeResponse
				err = json.NewDecoder(resp.Body).Decode(&meResponse)
				assert.NoError(t, err, "Failed to decode MeResponse")

				assert.Equal(t, currentUserDetails.UserID, meResponse.UserID)
				assert.Equal(t, currentUserDetails.Email, meResponse.Email)
				assert.Equal(t, currentUserDetails.Name, meResponse.UserName)
				assert.Equal(t, tt.expectedTenantName, meResponse.TenantName)

				// Check permission count based on user role
				if tt.expectedMinPermissions == 0 {
					// Editor should have exactly 0 permissions
					assert.Len(t, meResponse.Permissions, 0, "Editor should have no permissions")
				} else {
					// Admin should have at least the expected minimum permissions
					assert.GreaterOrEqual(t, len(meResponse.Permissions), tt.expectedMinPermissions,
						"Should have at least %d permissions", tt.expectedMinPermissions)
				}
			} else if tt.expectErrorResponse {
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received error response body for %s: %s", tt.name, string(bodyBytes)) // Log for debugging if needed
			}
		})
	}
}
