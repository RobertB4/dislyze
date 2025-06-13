package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestUpdateUserRoles_Integration(t *testing.T) {
	// Helper function to create a UUID from string
	mustParseUUID := func(s string) pgtype.UUID {
		var uuid pgtype.UUID
		err := uuid.Scan(s)
		if err != nil {
			t.Fatalf("Failed to parse UUID %s: %v", s, err)
		}
		return uuid
	}

	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                 string
		loginUserKey         string // Key for setup.TestUsersData map, empty for unauth
		targetUserKey        string // Key for setup.TestUsersData map of target user
		targetUserIDOverride string // Use this if targetUserKey is empty (for invalid userID tests)
		requestBody          users.UpdateUserRolesRequestBody
		expectedStatus       int
		expectUnauth         bool
		validateResponse     func(t *testing.T, resp *http.Response) // For custom response validation
	}{
		// Authentication & Authorization Tests
		{
			name:           "unauthenticated request gets 401",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "user without users.update permission gets 403 forbidden",
			loginUserKey:   "enterprise_2",
			targetUserKey:  "enterprise_11",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user from different tenant gets 403 forbidden",
			loginUserKey:   "smb_1",        // SMB tenant
			targetUserKey:  "enterprise_2", // Enterprise tenant
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:                 "invalid userID format gets 400",
			loginUserKey:         "enterprise_1",
			targetUserIDOverride: "not-a-uuid",
			requestBody:          users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus:       http.StatusBadRequest,
		},
		{
			name:           "empty role gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role value gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON request gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty request body gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "null role_ids field gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic Tests
		{
			name:                 "non-existent user gets 404",
			loginUserKey:         "enterprise_1",
			targetUserIDOverride: "00000000-0000-0000-0000-000000000000", // Valid UUID that doesn't exist
			requestBody:          users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus:       http.StatusNotFound,
		},
		{
			name:           "user trying to update own role gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_1", // Same user
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role IDs get 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("00000000-0000-0000-0000-000000000000")}}, // Non-existent role
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "roles from different tenant get 400",
			loginUserKey:   "enterprise_1",                                                                                                  // Enterprise admin
			targetUserKey:  "enterprise_2",                                                                                                  // Enterprise user
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")}}, // SMB admin role
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed UUID in role_ids gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty string UUID in role_ids gets 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate role IDs get 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}}, // Same role twice
			expectedStatus: http.StatusBadRequest,                                                                                                                                                  // Validation correctly rejects duplicates
		},
		{
			name:           "mixed valid and invalid role IDs get 400",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), mustParseUUID("00000000-0000-0000-0000-000000000000")}}, // Valid + invalid
			expectedStatus: http.StatusBadRequest,
		},

		// Success Tests
		{
			name:           "user with users.update permission successfully updates roles",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
		{
			name:           "successfully replaces existing roles",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2", // Was updated to admin in previous test
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
		{
			name:           "successfully assigns multiple roles",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
		{
			name:           "successfully sets same roles (no changes needed)",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2", // Should have both roles from previous test
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
		{
			name:           "successfully removes some roles (partial update)",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2", // Should have both roles, remove one
			requestBody:    users.UpdateUserRolesRequestBody{RoleIDs: []pgtype.UUID{mustParseUUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")}},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetUserID string
			if tt.targetUserKey != "" {
				targetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				assert.True(t, ok, "Target user key '%s' not found in setup.TestUsersData", tt.targetUserKey)
				targetUserID = targetUserDetails.UserID
			} else if tt.targetUserIDOverride != "" {
				targetUserID = tt.targetUserIDOverride
			} else {
				t.Fatal("Either targetUserKey or targetUserIDOverride must be provided")
			}

			var reqBody []byte
			var err error

			if tt.name == "malformed JSON request gets 400" {
				// Send malformed JSON for this specific test
				reqBody = []byte(`{"role": "admin", invalid}`)
			} else if tt.name == "empty request body gets 400" {
				// Send empty body
				reqBody = []byte(``)
			} else if tt.name == "null role_ids field gets 400" {
				// Send JSON with null role_ids
				reqBody = []byte(`{"role_ids": null}`)
			} else if tt.name == "malformed UUID in role_ids gets 400" {
				// Send JSON with malformed UUID string
				reqBody = []byte(`{"role_ids": ["not-a-valid-uuid-format"]}`)
			} else if tt.name == "empty string UUID in role_ids gets 400" {
				// Send JSON with empty string as UUID
				reqBody = []byte(`{"role_ids": [""]}`)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err, "Failed to marshal request body")
			}

			reqURL := fmt.Sprintf("%s/users/%s/roles", setup.BaseURL, targetUserID)
			req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqBody))
			assert.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Add authentication if not testing unauthenticated scenario
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key '%s' not found in setup.TestUsersData", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request")
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			// Custom response validation if provided
			if tt.validateResponse != nil {
				tt.validateResponse(t, resp)
			}

			// For successful updates, verify the roles were actually changed in database
			if tt.expectedStatus == http.StatusOK && tt.targetUserKey != "" {
				ctx := context.Background()

				// Query the user's current role IDs from user_roles table
				rows, err := pool.Query(ctx, `
					SELECT r.name 
					FROM user_roles ur 
					JOIN roles r ON ur.role_id = r.id 
					WHERE ur.user_id = $1
					ORDER BY r.name`, targetUserID)
				assert.NoError(t, err, "Failed to query updated user roles from database")
				defer rows.Close()

				var actualRoleNames []string
				for rows.Next() {
					var roleName string
					err := rows.Scan(&roleName)
					assert.NoError(t, err, "Failed to scan role name")
					actualRoleNames = append(actualRoleNames, roleName)
				}

				// Convert expected role IDs to role names for comparison
				expectedRoleNames := make([]string, len(tt.requestBody.RoleIDs))
				for i, roleID := range tt.requestBody.RoleIDs {
					var roleName string
					err := pool.QueryRow(ctx, "SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
					assert.NoError(t, err, "Failed to query expected role name")
					expectedRoleNames[i] = roleName
				}

				assert.ElementsMatch(t, expectedRoleNames, actualRoleNames, "Roles were not updated correctly in database")
			}
		})
	}
}
