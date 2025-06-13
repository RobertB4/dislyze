package roles

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"lugia/features/roles"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a role for testing by inserting directly into database
func createTestRole(t *testing.T, pool *pgxpool.Pool, name, description string, permissionIDs []string, tenantID string) string {
	ctx := context.Background()

	// Insert role into database
	var roleID string
	descriptionValue := sql.NullString{String: description, Valid: description != ""}

	err := pool.QueryRow(ctx,
		`INSERT INTO roles (tenant_id, name, description, is_default) 
		 VALUES ($1, $2, $3, false) 
		 RETURNING id`,
		tenantID, name, descriptionValue).Scan(&roleID)
	assert.NoError(t, err)

	// Insert role permissions if any
	for _, permissionID := range permissionIDs {
		_, err := pool.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id, tenant_id) 
			 VALUES ($1, $2, $3)`,
			roleID, permissionID, tenantID)
		assert.NoError(t, err)
	}

	return roleID
}

func TestUpdateRole_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	type updateRoleTestCase struct {
		name           string
		loginUserKey   string                                        // Key for setup.TestUsersData2 map, empty for unauth
		setupRole      func(t *testing.T, pool *pgxpool.Pool) string // Function to setup a role and return its ID
		roleID         string                                        // For tests that don't need setup
		requestBody    roles.UpdateRoleRequestBody
		expectedStatus int
		expectUnauth   bool
	}

	tests := []updateRoleTestCase{
		// Authentication & Authorization Tests
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			roleID:       "b0000000-0000-0000-0000-000000000001", // Any ID for unauth test
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Unauthorized Update",
				Description:   "This should fail",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "user without roles.update permission gets 403 forbidden",
			loginUserKey: "enterprise_2",
			roleID:       "b0000000-0000-0000-0000-000000000001", // Any ID for forbidden test
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Forbidden Update",
				Description:   "This should be forbidden",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusForbidden,
		},

		// Default Role Protection Tests
		{
			name:         "attempt to update default role returns 400",
			loginUserKey: "enterprise_1",
			roleID:       "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", // Default admin role from seed (is_default = true)
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Updated Default Role",
				Description:   "This should fail",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},

		// Input Validation Tests
		{
			name:         "invalid role ID format returns 400",
			loginUserKey: "enterprise_1",
			roleID:       "not-a-valid-uuid",
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid Description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "non-existent role ID returns 404",
			loginUserKey: "enterprise_1",
			roleID:       "99999999-9999-9999-9999-999999999999",
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid Description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "attempt to update role from different tenant returns 404",
			loginUserKey: "enterprise_1",                         // Enterprise tenant user
			roleID:       "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", // SMB tenant role from seed
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Cross Tenant Attack",
				Description:   "Should not be allowed",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "empty role name returns 400",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Empty Name", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "",
				Description:   "Valid Description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid permission ID format returns 500",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Invalid Permission", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid Description",
				PermissionIDs: []string{"not-a-valid-uuid"},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "non-existent permission ID returns 500",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Non-existent Permission", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid Description",
				PermissionIDs: []string{"99999999-9999-9999-9999-999999999999"},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "mix of valid and invalid permission IDs returns 500",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Mixed Permissions", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:        "Valid Name",
				Description: "Valid Description",
				PermissionIDs: []string{
					"3a52c807-ddcb-4044-8682-658e04800a8e", // Valid permission
					"99999999-9999-9999-9999-999999999999", // Invalid permission
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},

		// Name Uniqueness Tests
		{
			name:         "update role name to existing role name in same tenant returns 400",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				// Create two roles, try to update second to have first's name
				createTestRole(t, pool, "Existing Role Name", "First Role",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
				return createTestRole(t, pool, "Second Role Name", "Second Role",
					[]string{"db994eda-6ff7-4ae5-a675-3abe735ce9cc"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Existing Role Name",
				Description:   "Trying to use existing name",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "success: update role name to same name as role in different tenant",
			loginUserKey: "enterprise_1", // Enterprise tenant user
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				// First: Create a role in SMB tenant with name "Marketing Manager"
				createTestRole(t, pool, "Marketing Manager", "Marketing role in SMB tenant",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "22222222-2222-2222-2222-222222222222")

				// Second: Create a role in Enterprise tenant with different name to be updated
				return createTestRole(t, pool, "Sales Representative", "Sales role in Enterprise tenant",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Marketing Manager", // Same name as the role we created in SMB tenant
				Description:   "Cross tenant name should be allowed",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusOK,
		},

		// Edge Cases
		{
			name:         "role name exceeding 255 characters fails with DB error",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Long Name", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          strings.Repeat("a", 256), // 256 characters
				Description:   "Valid Description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "description exceeding 255 characters fails with DB error",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Test Role for Long Description", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Valid Name",
				Description:   strings.Repeat("b", 256), // 256 characters
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusInternalServerError,
		},

		// Happy Path Tests
		{
			name:         "success: update role with valid data",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Original Role", "Original Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Updated Role",
				Description:   "Updated Description",
				PermissionIDs: []string{"db994eda-6ff7-4ae5-a675-3abe735ce9cc"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: update role with empty permissions array",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Role with Permissions", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Role without Permissions",
				Description:   "Now has no permissions",
				PermissionIDs: []string{},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: update role with same name (no-op)",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Same Name Role", "Original Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Same Name Role",
				Description:   "Updated Description",
				PermissionIDs: []string{"db994eda-6ff7-4ae5-a675-3abe735ce9cc"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: update only description keeping same name",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Constant Name", "Original Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Constant Name",
				Description:   "New Description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: update role with empty description",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Role to Clear Description", "Will be cleared",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody: roles.UpdateRoleRequestBody{
				Name:          "Role with Empty Description",
				Description:   "",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusOK,
		},

		// Request Format Tests
		{
			name:         "malformed JSON body returns 400",
			loginUserKey: "enterprise_1",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRole(t, pool, "Role for Malformed JSON", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "11111111-1111-1111-1111-111111111111")
			},
			requestBody:    roles.UpdateRoleRequestBody{}, // Will be overridden in test
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var roleID string

			// Setup role if needed
			if tt.setupRole != nil {
				roleID = tt.setupRole(t, pool)
			} else {
				roleID = tt.roleID
			}

			// Prepare request
			var jsonData []byte
			var err error

			// Special case for malformed JSON test
			if tt.name == "malformed JSON body returns 400" {
				jsonData = []byte("{invalid json}")
			} else {
				jsonData, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			url := fmt.Sprintf("%s/roles/%s/update", setup.BaseURL, roleID)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Add authentication if needed
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData2[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData2: %s for test: %s", tt.loginUserKey, tt.name)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
				})
			}

			// Execute request
			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)

			defer resp.Body.Close()

			// Verify response status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Response status should match expected")

			// For successful updates, verify the response and check data integrity
			if resp.StatusCode == http.StatusOK {
				// Additional verification can be added here
				// For example, making a GET request to verify the role was updated correctly
			}
		})
	}
}
