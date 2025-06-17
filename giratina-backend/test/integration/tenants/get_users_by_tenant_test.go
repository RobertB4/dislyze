package tenants

import (
	"encoding/json"
	"fmt"
	"giratina/features/tenants"
	"giratina/test/integration/setup"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUsersByTenant_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                string
		tenantID            string
		loginUserKey        string // Key for setup.TestUsersData map
		expectedStatus      int
		expectErrorResponse bool
		expectedMinUsers    int // Minimum number of users expected
	}{
		{
			name:             "enterprise tenant has >= 100 users",
			tenantID:         "11111111-1111-1111-1111-111111111111", // Enterprise tenant
			loginUserKey:     "internal_1",
			expectedStatus:   http.StatusOK,
			expectedMinUsers: 100,
		},
		{
			name:             "SMB tenant has >= 10 users",
			tenantID:         "22222222-2222-2222-2222-222222222222", // SMB tenant
			loginUserKey:     "internal_1",
			expectedStatus:   http.StatusOK,
			expectedMinUsers: 10,
		},
		{
			name:                "invalid UUID returns 400",
			tenantID:            "invalid-uuid",
			loginUserKey:        "internal_1",
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:             "non-existent tenant returns empty list",
			tenantID:         "99999999-9999-9999-9999-999999999999", // Non-existent tenant
			loginUserKey:     "internal_1",
			expectedStatus:   http.StatusOK,
			expectedMinUsers: 0,
		},
		{
			name:                "unauthorized returns 401",
			tenantID:            "11111111-1111-1111-1111-111111111111",
			loginUserKey:        "", // No login
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cookies []*http.Cookie

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

			req, err := http.NewRequest("GET", fmt.Sprintf("%s/tenants/%s/users", setup.BaseURL, tt.tenantID), nil)
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
				var response tenants.GetUsersByTenantResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err, "Failed to decode GetUsersByTenantResponse")

				assert.GreaterOrEqual(t, len(response.Users), tt.expectedMinUsers,
					"Should have at least %d users", tt.expectedMinUsers)

			} else if tt.expectErrorResponse {
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received error response body for %s: %s", tt.name, string(bodyBytes))
			}
		})
	}
}