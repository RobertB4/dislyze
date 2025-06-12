package roles

import (
	"context"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a role for testing by inserting directly into database
func createTestRoleForDeletion(t *testing.T, pool *pgxpool.Pool, name, description string, permissionIDs []string, tenantID string) string {
	ctx := context.Background()

	// Insert role into database
	var roleID string
	err := pool.QueryRow(ctx,
		`INSERT INTO roles (tenant_id, name, description, is_default) 
		 VALUES ($1, $2, $3, false) 
		 RETURNING id`,
		tenantID, name, description).Scan(&roleID)
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

// Helper function to assign a role to a user
func assignRoleToUser(t *testing.T, pool *pgxpool.Pool, userID, roleID, tenantID string) {
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) 
		 VALUES ($1, $2, $3)`,
		userID, roleID, tenantID)
	assert.NoError(t, err)
}

// Helper function to verify role is deleted
func verifyRoleDeleted(t *testing.T, pool *pgxpool.Pool, roleID, tenantID string) {
	ctx := context.Background()

	// Check role is deleted
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM roles WHERE id = $1 AND tenant_id = $2`,
		roleID, tenantID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Role should be deleted")

	// Check role permissions are deleted
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM role_permissions WHERE role_id = $1 AND tenant_id = $2`,
		roleID, tenantID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Role permissions should be deleted")
}

func TestDeleteRole_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	type deleteRoleTestCase struct {
		name           string
		loginUserKey   string                                        // Key for setup.TestUsersData map, empty for unauth
		setupRole      func(t *testing.T, pool *pgxpool.Pool) string // Function to setup a role and return its ID
		roleID         string                                        // For tests that don't need setup
		expectedStatus int
		expectUnauth   bool
		verifyDeleted  bool // Whether to verify the role was actually deleted
	}

	tests := []deleteRoleTestCase{
		// Authentication & Authorization Tests
		{
			name:           "error for unauthorized request",
			expectUnauth:   true,
			roleID:         "b0000000-0000-0000-0000-000000000001", // Any ID for unauth test
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "user without roles.edit permission gets 403 forbidden",
			loginUserKey:   "alpha_editor",
			roleID:         "b0000000-0000-0000-0000-000000000001", // Any ID for forbidden test
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:           "invalid role ID format returns 400",
			loginUserKey:   "alpha_admin",
			roleID:         "not-a-valid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-existent role ID returns 404",
			loginUserKey:   "alpha_admin",
			roleID:         "99999999-9999-9999-9999-999999999999",
			expectedStatus: http.StatusNotFound,
		},

		// Business Logic Protection Tests
		{
			name:           "cannot delete default admin role",
			loginUserKey:   "alpha_admin",
			roleID:         "e0000000-0000-0000-0000-000000000001", // Default admin role from seed
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "cannot delete default editor role",
			loginUserKey:   "alpha_admin",
			roleID:         "e0000000-0000-0000-0000-000000000002", // Default editor role from seed
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "cannot delete role assigned to users",
			loginUserKey: "alpha_admin",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				// Create a role and assign it to a user
				roleID := createTestRoleForDeletion(t, pool, "Role With Users", "Test Description",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "a0000000-0000-0000-0000-000000000001")
				// Assign to alpha_admin user
				assignRoleToUser(t, pool, "b0000000-0000-0000-0000-000000000001", roleID, "a0000000-0000-0000-0000-000000000001")
				return roleID
			},
			expectedStatus: http.StatusBadRequest,
		},

		// Security & Tenant Isolation Tests
		{
			name:           "attempt to delete role from different tenant returns 404",
			loginUserKey:   "alpha_admin",                          // Tenant Alpha user
			roleID:         "e0000000-0000-0000-0000-000000000003", // Tenant Beta role from seed
			expectedStatus: http.StatusNotFound,
		},

		// Success Cases
		{
			name:         "successfully delete custom role with no users assigned",
			loginUserKey: "alpha_admin",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRoleForDeletion(t, pool, "Deletable Role", "Can be deleted",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "a0000000-0000-0000-0000-000000000001")
			},
			expectedStatus: http.StatusOK,
			verifyDeleted:  true,
		},
		{
			name:         "successfully delete role with permissions but no users",
			loginUserKey: "alpha_admin",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRoleForDeletion(t, pool, "Role With Permissions", "Has permissions but no users",
					[]string{
						"3a52c807-ddcb-4044-8682-658e04800a8e", // users.view
						"db994eda-6ff7-4ae5-a675-3abe735ce9cc", // users.edit
					}, "a0000000-0000-0000-0000-000000000001")
			},
			expectedStatus: http.StatusOK,
			verifyDeleted:  true,
		},
		{
			name:         "successfully delete role with empty permission set",
			loginUserKey: "alpha_admin",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRoleForDeletion(t, pool, "Role No Permissions", "No permissions assigned",
					[]string{}, "a0000000-0000-0000-0000-000000000001")
			},
			expectedStatus: http.StatusOK,
			verifyDeleted:  true,
		},

		// Edge Cases
		{
			name:         "delete role and verify it's really gone",
			loginUserKey: "alpha_admin",
			setupRole: func(t *testing.T, pool *pgxpool.Pool) string {
				return createTestRoleForDeletion(t, pool, "To Be Verified Gone", "This role will be verified as deleted",
					[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "a0000000-0000-0000-0000-000000000001")
			},
			expectedStatus: http.StatusOK,
			verifyDeleted:  true,
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
			url := fmt.Sprintf("%s/roles/%s/delete", setup.BaseURL, roleID)
			req, err := http.NewRequest("POST", url, nil)
			assert.NoError(t, err)

			// Add authentication if needed
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s for test: %s", tt.loginUserKey, tt.name)

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
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			// Verify response status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Test: %s", tt.name)

			// Verify role is actually deleted for success cases
			if tt.verifyDeleted && resp.StatusCode == http.StatusOK {
				verifyRoleDeleted(t, pool, roleID, "a0000000-0000-0000-0000-000000000001")
			}
		})
	}
}

// Additional test to verify deletion and subsequent access
func TestDeleteRole_VerifyGoneAfterDeletion(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	// Create a test role
	roleID := createTestRoleForDeletion(t, pool, "Role To Delete And Verify", "Will be deleted",
		[]string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, "a0000000-0000-0000-0000-000000000001")

	// Get auth token
	loginDetails, ok := setup.TestUsersData["alpha_admin"]
	assert.True(t, ok)
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	// Delete the role
	deleteURL := fmt.Sprintf("%s/roles/%s/delete", setup.BaseURL, roleID)
	deleteReq, err := http.NewRequest("POST", deleteURL, nil)
	assert.NoError(t, err)
	deleteReq.AddCookie(&http.Cookie{
		Name:  "dislyze_access_token",
		Value: accessToken,
	})

	client := &http.Client{}
	deleteResp, err := client.Do(deleteReq)
	assert.NoError(t, err)
	defer func() {
		if err := deleteResp.Body.Close(); err != nil {
			t.Logf("Failed to close delete response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

	// Try to delete the same role again - should return 404
	deleteReq2, err := http.NewRequest("POST", deleteURL, nil)
	assert.NoError(t, err)
	deleteReq2.AddCookie(&http.Cookie{
		Name:  "dislyze_access_token",
		Value: accessToken,
	})

	deleteResp2, err := client.Do(deleteReq2)
	assert.NoError(t, err)
	defer func() {
		if err := deleteResp2.Body.Close(); err != nil {
			t.Logf("Failed to close second delete response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusNotFound, deleteResp2.StatusCode, "Attempting to delete already deleted role should return 404")
}
