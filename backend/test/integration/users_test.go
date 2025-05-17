package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"lugia/handlers"
	"lugia/test/integration/setup"
)

func TestGetUsers_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name               string
		loginUserKey       string // Key for setup.TestUsersData map
		expectedStatus     int
		expectedUserEmails []string
		expectUnauth       bool
	}{
		{
			name:           "unauthenticated user gets 401",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "alpha_admin (Tenant A) gets users from Tenant Alpha",
			loginUserKey:   "alpha_admin",
			expectedStatus: http.StatusOK,
			// Order by created_at ASC from setup.sql: alpha_admin (11:00) then alpha_user (11:01)
			expectedUserEmails: []string{setup.TestUsersData["alpha_admin"].Email, setup.TestUsersData["alpha_user"].Email},
		},
		{
			name:               "alpha_user (Tenant A) gets users from Tenant Alpha",
			loginUserKey:       "alpha_user",
			expectedStatus:     http.StatusOK,
			expectedUserEmails: []string{setup.TestUsersData["alpha_admin"].Email, setup.TestUsersData["alpha_user"].Email},
		},
		{
			name:               "beta_admin (Tenant B) gets users from Tenant Beta (only self)",
			loginUserKey:       "beta_admin",
			expectedStatus:     http.StatusOK,
			expectedUserEmails: []string{setup.TestUsersData["beta_admin"].Email},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/users", setup.BaseURL), nil)
			assert.NoError(t, err)

			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				// Assuming your auth middleware expects a cookie named "access_token"
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse []handlers.User
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				assert.Equal(t, len(tt.expectedUserEmails), len(usersResponse), "Number of users mismatch for test: %s", tt.name)

				actualEmails := make([]string, len(usersResponse))
				for i, u := range usersResponse {
					actualEmails[i] = u.Email
					assert.NotEmpty(t, u.ID, "User ID should not be empty for user %s", u.Email)
					assert.Equal(t, "active", u.Status, "User status should be active for user %s", u.Email)

					var expectedName, expectedRole, expectedUserID string
					foundInTestData := false
					for _, seededUser := range setup.TestUsersData {
						if seededUser.Email == u.Email {
							expectedName = seededUser.Name
							expectedRole = seededUser.Role
							expectedUserID = seededUser.UserID
							foundInTestData = true
							break
						}
					}
					assert.True(t, foundInTestData, "User with email %s not found in setup.TestUsersData. Check setup.sql and setup.TestUsersData map.", u.Email)
					assert.Equal(t, expectedUserID, u.ID, "ID mismatch for user %s", u.Email)
					assert.Equal(t, expectedName, u.Name, "Name mismatch for user %s", u.Email)
					assert.Equal(t, expectedRole, u.Role, "Role mismatch for user %s", u.Email)
				}
				assert.Equal(t, tt.expectedUserEmails, actualEmails, "User email list or order mismatch for test: %s", tt.name)
			}
		})
	}
}
