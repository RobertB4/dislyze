package users

import (
	"encoding/json"
	"fmt"

	"lugia/features/users"
	"lugia/queries_pregeneration"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUsers_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
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
			// Order by created_at DESC from seed.sql
			expectedUserEmails: []string{
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:           "alpha_editor (Tenant A) gets forbidden because they are not an admin",
			loginUserKey:   "alpha_editor",
			expectedStatus: http.StatusForbidden,
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
			reqURL := fmt.Sprintf("%s/users?page=1&limit=50", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
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
				var usersResponse users.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, 1, usersResponse.Pagination.Page, "Pagination page mismatch for test: %s", tt.name)
				assert.Equal(t, 50, usersResponse.Pagination.Limit, "Pagination limit mismatch for test: %s", tt.name)
				assert.Equal(t, len(tt.expectedUserEmails), usersResponse.Pagination.Total, "Pagination total mismatch for test: %s", tt.name)

				assert.Equal(t, len(tt.expectedUserEmails), len(usersResponse.Users), "Number of users mismatch for test: %s", tt.name)

				actualEmails := make([]string, len(usersResponse.Users))
				for i, u := range usersResponse.Users {
					actualEmails[i] = u.Email
					assert.NotEmpty(t, u.ID, "User ID should not be empty for user %s", u.Email)

					var expectedName, expectedUserID, expectedStatus string
					var expectedRole queries_pregeneration.UserRole
					foundInTestData := false
					for _, seededUser := range setup.TestUsersData {
						if seededUser.Email == u.Email {
							expectedName = seededUser.Name
							expectedRole = seededUser.Role
							expectedUserID = seededUser.UserID
							expectedStatus = seededUser.Status
							foundInTestData = true
							break
						}
					}
					assert.True(t, foundInTestData, "User with email %s not found in setup.TestUsersData. Check setup.sql and setup.TestUsersData map.", u.Email)
					assert.Equal(t, expectedUserID, u.ID, "ID mismatch for user %s", u.Email)
					assert.Equal(t, expectedName, u.Name, "Name mismatch for user %s", u.Email)
					assert.Equal(t, expectedRole, u.Role, "Role mismatch for user %s", u.Email)
					assert.Equal(t, expectedStatus, u.Status, "Status mismatch for user %s", u.Email)
				}
				assert.Equal(t, tt.expectedUserEmails, actualEmails, "User email list or order mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestGetUsersPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name               string
		page               int
		limit              int
		expectedStatus     int
		expectedPage       int
		expectedLimit      int
		expectedTotal      int
		expectedTotalPages int
		expectedHasNext    bool
		expectedHasPrev    bool
		expectedUserCount  int
	}{
		{
			name:               "page 1 with limit 2 - first page",
			page:               1,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      2,
			expectedTotal:      6, // Total users in Tenant A
			expectedTotalPages: 3, // 6 users / 2 per page = 3 pages
			expectedHasNext:    true,
			expectedHasPrev:    false,
			expectedUserCount:  2,
		},
		{
			name:               "page 2 with limit 2 - middle page",
			page:               2,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       2,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    true,
			expectedHasPrev:    true,
			expectedUserCount:  2,
		},
		{
			name:               "page 3 with limit 2 - last page",
			page:               3,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       3,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    false,
			expectedHasPrev:    true,
			expectedUserCount:  2,
		},
		{
			name:               "page beyond total pages returns empty results",
			page:               5,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       5,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    false,
			expectedHasPrev:    true,
			expectedUserCount:  0,
		},
		{
			name:               "large limit gets all users in one page",
			page:               1,
			limit:              10,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      10,
			expectedTotal:      6,
			expectedTotalPages: 1,
			expectedHasNext:    false,
			expectedHasPrev:    false,
			expectedUserCount:  6,
		},
		{
			name:               "limit exceeding max (100) gets capped",
			page:               1,
			limit:              150,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      100, // Should be capped at 100
			expectedTotal:      6,
			expectedTotalPages: 1,
			expectedHasNext:    false,
			expectedHasPrev:    false,
			expectedUserCount:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=%d&limit=%d", setup.BaseURL, tt.page, tt.limit)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse users.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedLimit, usersResponse.Pagination.Limit, "Limit mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotal, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotalPages, usersResponse.Pagination.TotalPages, "TotalPages mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasNext, usersResponse.Pagination.HasNext, "HasNext mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasPrev, usersResponse.Pagination.HasPrev, "HasPrev mismatch for test: %s", tt.name)

				// Verify user count
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestGetUsersSearch_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name                string
		search              string
		expectedStatus      int
		expectedUserCount   int
		expectedContains    []string // Emails that should be in results
		expectedNotContains []string // Emails that should not be in results
	}{
		{
			name:              "search by name 'Admin' finds admin users",
			search:            "Admin",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:              "search by name 'Editor' finds editor users",
			search:            "Editor",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 5, // alpha_editor, pending_editor_valid_token, suspended_editor, pending_editor_for_rate_limit_test, pending_editor_tenant_A_for_x_tenant_test
			expectedContains: []string{
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:              "search by partial name 'Pending' finds pending users",
			search:            "Pending",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 3, // All pending users
			expectedContains: []string{
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
			},
		},
		{
			name:              "search by email domain 'alpha' finds alpha users",
			search:            "alpha",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 2,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:              "case insensitive search 'ADMIN' finds admin",
			search:            "ADMIN",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:              "search for 'Suspended' finds suspended user",
			search:            "Suspended",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["suspended_editor"].Email,
			},
		},
		{
			name:              "search for nonexistent term returns empty results",
			search:            "nonexistent",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 0,
			expectedContains:  []string{},
		},
		{
			name:              "empty search returns all users",
			search:            "",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 6, // All users in Tenant A
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=1&limit=50&search=%s", setup.BaseURL, tt.search)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse users.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify user count
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)

				// Verify total in pagination matches user count for these tests
				assert.Equal(t, tt.expectedUserCount, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)

				// Collect actual emails
				actualEmails := make([]string, len(usersResponse.Users))
				for i, user := range usersResponse.Users {
					actualEmails[i] = user.Email
				}

				// Verify expected emails are present
				for _, expectedEmail := range tt.expectedContains {
					assert.Contains(t, actualEmails, expectedEmail, "Expected email %s not found in results for test: %s", expectedEmail, tt.name)
				}

				// Verify unexpected emails are not present
				for _, unexpectedEmail := range tt.expectedNotContains {
					assert.NotContains(t, actualEmails, unexpectedEmail, "Unexpected email %s found in results for test: %s", unexpectedEmail, tt.name)
				}
			}
		})
	}
}

func TestGetUsersSearchWithPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name              string
		search            string
		page              int
		limit             int
		expectedStatus    int
		expectedTotal     int
		expectedUserCount int
		expectedPage      int
		expectedHasNext   bool
		expectedHasPrev   bool
	}{
		{
			name:              "search 'Editor' with pagination - page 1 limit 2",
			search:            "Editor",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     5, // 5 users with "Editor" in name
			expectedUserCount: 2, // First 2 results
			expectedPage:      1,
			expectedHasNext:   true,
			expectedHasPrev:   false,
		},
		{
			name:              "search 'Editor' with pagination - page 2 limit 2",
			search:            "Editor",
			page:              2,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     5,
			expectedUserCount: 2, // Next 2 results
			expectedPage:      2,
			expectedHasNext:   true, // Still more results (page 3 will have 1 result)
			expectedHasPrev:   true,
		},
		{
			name:              "search 'Admin' with pagination - single result",
			search:            "Admin",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     1, // Only 1 admin
			expectedUserCount: 1,
			expectedPage:      1,
			expectedHasNext:   false,
			expectedHasPrev:   false,
		},
		{
			name:              "search 'nonexistent' with pagination - no results",
			search:            "nonexistent",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     0,
			expectedUserCount: 0,
			expectedPage:      1,
			expectedHasNext:   false,
			expectedHasPrev:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=%d&limit=%d&search=%s", setup.BaseURL, tt.page, tt.limit, tt.search)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse users.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotal, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasNext, usersResponse.Pagination.HasNext, "HasNext mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasPrev, usersResponse.Pagination.HasPrev, "HasPrev mismatch for test: %s", tt.name)

				// Verify all returned users match the search term
				for _, user := range usersResponse.Users {
					nameMatch := strings.Contains(strings.ToLower(user.Name), strings.ToLower(tt.search))
					emailMatch := strings.Contains(strings.ToLower(user.Email), strings.ToLower(tt.search))
					if tt.search != "" && tt.search != "nonexistent" {
						assert.True(t, nameMatch || emailMatch,
							"User %s (%s) does not match search term '%s' for test: %s",
							user.Name, user.Email, tt.search, tt.name)
					}
				}
			}
		})
	}
}

func TestGetUsersInvalidParameters_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to users
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedPage   int
		expectedLimit  int
	}{
		{
			name:           "invalid page parameter - non-numeric defaults to 1",
			queryParams:    "page=abc&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "invalid limit parameter - non-numeric defaults to 50",
			queryParams:    "page=1&limit=xyz",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "negative page parameter defaults to 1",
			queryParams:    "page=-1&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "zero page parameter defaults to 1",
			queryParams:    "page=0&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "negative limit parameter defaults to 50",
			queryParams:    "page=1&limit=-5",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "zero limit parameter defaults to 50",
			queryParams:    "page=1&limit=0",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "missing parameters use defaults",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "only page parameter provided",
			queryParams:    "page=2",
			expectedStatus: http.StatusOK,
			expectedPage:   2,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "only limit parameter provided",
			queryParams:    "limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users", setup.BaseURL)
			if tt.queryParams != "" {
				reqURL += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s", tt.name)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse users.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify the exact expected default values are applied
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page should match expected default for test: %s", tt.name)
				assert.Equal(t, tt.expectedLimit, usersResponse.Pagination.Limit, "Limit should match expected default for test: %s", tt.name)

				// Additional validation to ensure reasonable values
				assert.True(t, usersResponse.Pagination.Page >= 1, "Page should be at least 1")
				assert.True(t, usersResponse.Pagination.Limit >= 1, "Limit should be at least 1")
				assert.True(t, usersResponse.Pagination.Limit <= 100, "Limit should not exceed 100")
			}
		})
	}
}
