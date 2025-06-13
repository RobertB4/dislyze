package users

import (
	"encoding/json"
	"fmt"

	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to verify tenant isolation
func validateTenantIsolation(t *testing.T, users []users.UserInfo, expectedTenantID string) {
	// This validates that all returned users belong to the expected tenant
	// We can't easily check the actual tenant_id from the API response, but we can check
	// that the user belongs to the same tenant by ensuring consistent behavior

	// For basic validation, we verify that no users from other known tenants are returned
	// This is a simplified check - in a full test environment, you might query the database directly
	for _, user := range users {
		// Verify that SMB tenant users are not returned when logged in as Enterprise user
		if expectedTenantID == "11111111-1111-1111-1111-111111111111" { // Enterprise tenant
			assert.False(t, strings.Contains(user.Email, "smb"),
				"Enterprise user should not see SMB user %s", user.Email)
			assert.False(t, strings.Contains(user.Email, "internal"),
				"Enterprise user should not see internal user %s", user.Email)
		}

		// Verify that Enterprise tenant users are not returned when logged in as SMB user
		if expectedTenantID == "22222222-2222-2222-2222-222222222222" { // SMB tenant
			assert.False(t, strings.Contains(user.Email, "enterprise"),
				"SMB user should not see Enterprise user %s", user.Email)
			assert.False(t, strings.Contains(user.Email, "internal"),
				"SMB user should not see internal user %s", user.Email)
		}
	}
}

func TestGetUsers_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData map
		expectedStatus int
		expectUnauth   bool
		validateFunc   func(t *testing.T, response users.GetUsersResponse, loginUser setup.UserTestData)
	}{
		{
			name:           "unauthenticated user gets 401",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "user with users.view permission successfully retrieves user list (Enterprise tenant)",
			loginUserKey:   "enterprise_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, loginUser setup.UserTestData) {
				// Verify tenant isolation - no users from other tenants should be returned
				validateTenantIsolation(t, response.Users, loginUser.TenantID)

				// Verify we got some users (Enterprise tenant should have 100+ users)
				assert.Greater(t, len(response.Users), 0, "Should return at least some users")
				assert.Equal(t, len(response.Users), response.Pagination.Limit, "Pagination limit should match returned user count")

				// Verify all users are from enterprise domain (specific to this tenant's test data)
				for _, user := range response.Users {
					assert.Contains(t, user.Email, "enterprise", "All users should be from enterprise domain")
				}
			},
		},
		{
			name:           "user without users.view permission gets 403 forbidden",
			loginUserKey:   "enterprise_2",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user with users.view permission only sees users from own tenant (SMB tenant)",
			loginUserKey:   "smb_1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, loginUser setup.UserTestData) {
				// Verify tenant isolation - no users from other tenants should be returned
				validateTenantIsolation(t, response.Users, loginUser.TenantID)

				// Verify we got some users (SMB tenant should have 10 users)
				assert.Greater(t, len(response.Users), 0, "Should return at least some users")
				assert.Equal(t, len(response.Users), response.Pagination.Total, "Pagination total should match returned user count")

				// Verify all users are from SMB domain (specific to this tenant's test data)
				for _, user := range response.Users {
					assert.Contains(t, user.Email, "smb", "All users should be from SMB domain")
				}
			},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=1&limit=50", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			var loginUser setup.UserTestData
			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData2[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData2: %s", tt.loginUserKey)
				loginUser = loginDetails

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

				// Verify all returned users have proper structure
				for _, u := range usersResponse.Users {
					assert.NotEmpty(t, u.ID, "User ID should not be empty for user %s", u.Email)
					assert.NotEmpty(t, u.Email, "User email should not be empty")
					assert.NotEmpty(t, u.Name, "User name should not be empty for user %s", u.Email)
					assert.NotEmpty(t, u.Status, "User status should not be empty for user %s", u.Email)

					// Verify user has at least one role with proper structure
					assert.NotEmpty(t, u.Roles, "User %s should have at least one role", u.Email)
					for _, role := range u.Roles {
						assert.NotEmpty(t, role.ID, "Role ID should not be empty for user %s", u.Email)
						assert.NotEmpty(t, role.Name, "Role name should not be empty for user %s", u.Email)
						assert.NotEmpty(t, role.Description, "Role description should not be empty for user %s", u.Email)

						// Verify role description matches known patterns
						if role.Name == "管理者" {
							assert.Equal(t, "すべての機能にアクセス可能", role.Description, "Admin role description mismatch for user %s", u.Email)
						} else if role.Name == "編集者" {
							assert.Equal(t, "ユーザー管理以外の編集権限", role.Description, "Editor role description mismatch for user %s", u.Email)
						}
					}
				}

				// Run custom validation if provided
				if tt.validateFunc != nil {
					tt.validateFunc(t, usersResponse, loginUser)
				}
			}
		})
	}
}

func TestGetUsersPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Use enterprise_1 who has access to users in Enterprise tenant
	loginDetails := setup.TestUsersData2["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	// First, get the total count of users by making a request with a large limit
	firstReq, err := http.NewRequest("GET", fmt.Sprintf("%s/users?page=1&limit=100", setup.BaseURL), nil)
	assert.NoError(t, err)
	firstReq.AddCookie(&http.Cookie{
		Name:  "dislyze_access_token",
		Value: accessToken,
		Path:  "/",
	})

	firstResp, err := client.Do(firstReq)
	assert.NoError(t, err)
	defer firstResp.Body.Close()

	var usersResponse users.GetUsersResponse
	err = json.NewDecoder(firstResp.Body).Decode(&usersResponse)
	assert.NoError(t, err)

	totalUsers := usersResponse.Pagination.Total
	assert.Greater(t, totalUsers, 0, "Should have at least some users in database")

	// Calculate pagination values dynamically
	limit2 := 2
	totalPages2 := (totalUsers + limit2 - 1) / limit2 // Ceiling division
	lastPageCount2 := totalUsers % limit2
	if lastPageCount2 == 0 {
		lastPageCount2 = limit2
	}

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
			limit:              limit2,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      limit2,
			expectedTotal:      totalUsers,
			expectedTotalPages: totalPages2,
			expectedHasNext:    totalPages2 > 1,
			expectedHasPrev:    false,
			expectedUserCount: func() int {
				if totalUsers >= limit2 {
					return limit2
				}
				return totalUsers
			}(),
		},
		{
			name:               "page 2 with limit 2 - middle page (if exists)",
			page:               2,
			limit:              limit2,
			expectedStatus:     http.StatusOK,
			expectedPage:       2,
			expectedLimit:      limit2,
			expectedTotal:      totalUsers,
			expectedTotalPages: totalPages2,
			expectedHasNext:    totalPages2 > 2,
			expectedHasPrev:    true,
			expectedUserCount: func() int {
				if totalPages2 < 2 {
					return 0 // No page 2
				} else if totalPages2 == 2 {
					return lastPageCount2 // Last page
				} else {
					return limit2 // Full page
				}
			}(),
		},
		{
			name:               "page beyond total pages returns empty results",
			page:               totalPages2 + 2,
			limit:              limit2,
			expectedStatus:     http.StatusOK,
			expectedPage:       totalPages2 + 2,
			expectedLimit:      limit2,
			expectedTotal:      totalUsers,
			expectedTotalPages: totalPages2,
			expectedHasNext:    false,
			expectedHasPrev:    true,
			expectedUserCount:  0,
		},
		{
			name:               "limit exceeding max (100) gets capped",
			page:               1,
			limit:              150,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      100, // Should be capped at 100
			expectedTotal:      totalUsers,
			expectedTotalPages: 2, // 100+ users but less than 200
			expectedHasNext:    true,
			expectedHasPrev:    false,
			expectedUserCount:  100,
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
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Use enterprise_1 who has access to users in Enterprise tenant
	loginDetails := setup.TestUsersData2["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	// We'll validate search functionality dynamically rather than comparing to static test data

	client := &http.Client{}

	tests := []struct {
		name           string
		search         string
		expectedStatus int
		validateFunc   func(t *testing.T, response users.GetUsersResponse, searchTerm string)
	}{
		{
			name:           "search functionality works with name pattern",
			search:         "田", // Common character in Japanese names that should match some users
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, searchTerm string) {
				// Verify search returns some results (田 should match several Japanese names)
				assert.Greater(t, len(response.Users), 0, "Search should return some matching users")
				assert.Equal(t, len(response.Users), response.Pagination.Total, "Pagination total should match search results")

				// Verify all returned users match the search term
				for _, user := range response.Users {
					nameMatch := strings.Contains(strings.ToLower(user.Name), strings.ToLower(searchTerm))
					emailMatch := strings.Contains(strings.ToLower(user.Email), strings.ToLower(searchTerm))
					assert.True(t, nameMatch || emailMatch,
						"User %s (%s) should match search term '%s'", user.Name, user.Email, searchTerm)
				}
			},
		},
		{
			name:           "search by common email domain pattern",
			search:         "localhost", // Most test emails contain localhost
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, searchTerm string) {
				// localhost should match many/all users since test emails contain localhost
				assert.Greater(t, len(response.Users), 0, "Search should find users with localhost in email")

				// All results should contain the search term in email
				for _, user := range response.Users {
					assert.Contains(t, strings.ToLower(user.Email), strings.ToLower(searchTerm),
						"User email %s should contain search term '%s'", user.Email, searchTerm)
				}
			},
		},
		{
			name:           "search for nonexistent term returns empty results",
			search:         "xyz_nonexistent_term_xyz",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, searchTerm string) {
				assert.Equal(t, 0, len(response.Users), "Search for nonexistent term should return no results")
				assert.Equal(t, 0, response.Pagination.Total, "Pagination total should be 0 for nonexistent term")
			},
		},
		{
			name:           "empty search returns all users",
			search:         "",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse, searchTerm string) {
				// Empty search should return all users in tenant (should be > 0)
				assert.Greater(t, len(response.Users), 0, "Empty search should return all tenant users")
				assert.Equal(t, len(response.Users), response.Pagination.Limit, "Pagination limit should match returned users")
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

				// Verify basic response structure
				assert.NotNil(t, usersResponse.Users, "Users array should not be nil")
				assert.NotNil(t, usersResponse.Pagination, "Pagination should not be nil")

				// Run custom validation
				if tt.validateFunc != nil {
					tt.validateFunc(t, usersResponse, tt.search)
				}
			}
		})
	}
}

func TestGetUsersSearchWithPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Use enterprise_1 who has access to users in Enterprise tenant
	loginDetails := setup.TestUsersData2["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name           string
		search         string
		page           int
		limit          int
		expectedStatus int
		validateFunc   func(t *testing.T, response users.GetUsersResponse)
	}{
		{
			name:           "search with pagination - basic test",
			search:         "localhost", // Most test emails contain this
			page:           1,
			limit:          2,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response users.GetUsersResponse) {
				// Basic validation - pagination structure is correct
				assert.Equal(t, 1, response.Pagination.Page, "Page should be 1")
				assert.Equal(t, 2, response.Pagination.Limit, "Limit should be 2")
				assert.True(t, response.Pagination.Total >= 0, "Total should be non-negative")
				assert.True(t, len(response.Users) <= 2, "Should not return more than limit")
			},
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

				// Run custom validation
				if tt.validateFunc != nil {
					tt.validateFunc(t, usersResponse)
				}
			}
		})
	}
}

func TestGetUsersInvalidParameters_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB2(t, pool)
	defer setup.CloseDB(pool)

	// Use enterprise_1 who has access to users
	loginDetails := setup.TestUsersData2["enterprise_1"]
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
