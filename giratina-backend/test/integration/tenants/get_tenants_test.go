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

func TestGetTenants_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                string
		loginUserKey        string // Key for setup.TestUsersData map
		expectedStatus      int
		expectErrorResponse bool
		expectedMinTenants  int // Minimum number of tenants expected
	}{
		{
			name:               "internal_1 gets tenants list",
			loginUserKey:       "internal_1",
			expectedStatus:     http.StatusOK,
			expectedMinTenants: 1, // At least one tenant should exist
		},
		{
			name:               "internal_2 gets tenants list",
			loginUserKey:       "internal_2",
			expectedStatus:     http.StatusOK,
			expectedMinTenants: 1,
		},
		{
			name:                "enterprise_1 gets 401 because they are not an internal admin",
			loginUserKey:        "enterprise_1",
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
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

			if !tt.expectErrorResponse {
				var ok bool
				currentUserDetails, ok = setup.TestUsersData[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: User key '%s' not found in TestUsersData", tt.loginUserKey)
				}

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			req, err := http.NewRequest("GET", fmt.Sprintf("%s/tenants", setup.BaseURL), nil)
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
				var tenantsResponse tenants.GetTenantsResponse
				err = json.NewDecoder(resp.Body).Decode(&tenantsResponse)
				assert.NoError(t, err, "Failed to decode GetTenantsResponse")

				assert.GreaterOrEqual(t, len(tenantsResponse.Tenants), tt.expectedMinTenants,
					"Should have at least %d tenants", tt.expectedMinTenants)

				// Validate tenant structure
				for _, tenant := range tenantsResponse.Tenants {
					assert.NotEmpty(t, tenant.ID, "Tenant ID should not be empty")
					assert.NotEmpty(t, tenant.Name, "Tenant name should not be empty")
					assert.NotEmpty(t, tenant.CreatedAt, "Tenant created_at should not be empty")
					assert.NotEmpty(t, tenant.UpdatedAt, "Tenant updated_at should not be empty")
					assert.NotNil(t, tenant.EnterpriseFeatures, "Enterprise features should not be nil")
				}

			} else if tt.expectErrorResponse {
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received error response body for %s: %s", tt.name, string(bodyBytes)) // Log for debugging if needed
			}
		})
	}
}