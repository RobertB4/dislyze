package roles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/roles"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestCreateRole_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB2(t, pool)

	type createRoleTestCase struct {
		name           string
		loginUserKey   string // Key for setup.TestUsersData2 map, empty for unauth
		requestBody    roles.CreateRoleRequestBody
		expectedStatus int
		expectUnauth   bool
	}

	tests := []createRoleTestCase{
		// Authentication & Authorization Tests
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Unauthorized Role",
				Description:   "This should fail",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "user without roles.create permission gets 403 forbidden",
			loginUserKey: "enterprise_2",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Forbidden Role",
				Description:   "This should be forbidden",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
			},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:         "validation error: missing name",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "",
				Description:   "Valid description",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "validation error: no permissions",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Valid Name",
				Description:   "Valid description",
				PermissionIDs: []string{}, // Empty permissions array
			},
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic Tests
		{
			name:         "error: duplicate role name in same tenant",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "管理者", // Same as existing admin role in Enterprise tenant
				Description:   "Duplicate name role",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "error: invalid permission ID",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Role with Invalid Permission",
				Description:   "This has an invalid permission",
				PermissionIDs: []string{"99999999-9999-9999-9999-999999999999"}, // Non-existent permission
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "error: malformed permission UUID",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Role with Malformed UUID",
				Description:   "This has a malformed permission ID",
				PermissionIDs: []string{"not-a-valid-uuid"},
			},
			expectedStatus: http.StatusInternalServerError,
		},

		// Success Tests
		{
			name:         "success: create role with single permission",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "Test Role Single",
				Description:   "A test role with one permission",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: create role with multiple permissions",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:        "Test Role Multi",
				Description: "A test role with multiple permissions",
				PermissionIDs: []string{
					"3a52c807-ddcb-4044-8682-658e04800a8e", // users.view
					"db994eda-6ff7-4ae5-a675-3abe735ce9cc", // users.edit
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "success: create role with empty description",
			loginUserKey: "enterprise_1",
			requestBody: roles.CreateRoleRequestBody{
				Name:          "No Description Role",
				Description:   "",
				PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			jsonData, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", setup.BaseURL+"/roles/create", bytes.NewBuffer(jsonData))
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
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Failed to close response body: %v", err)
				}
			}()

			// Verify response status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Test: %s", tt.name)
		})
	}
}

// createTenantViaSignup creates a new tenant and admin user via the signup endpoint
// Returns the user credentials for authentication
func createTenantViaSignup(t *testing.T, email, password, tenantName, userName string) struct {
	Email    string
	Password string
	TenantID string
	UserID   string
} {
	// Prepare signup request
	signupRequest := map[string]string{
		"email":            email,
		"password":         password,
		"password_confirm": password,
		"company_name":     tenantName,
		"user_name":        userName,
	}

	jsonData, err := json.Marshal(signupRequest)
	assert.NoError(t, err)

	// Make signup request
	req, err := http.NewRequest("POST", setup.BaseURL+"/auth/signup", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Signup should succeed
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Signup should succeed")

	return struct {
		Email    string
		Password string
		TenantID string
		UserID   string
	}{
		Email:    email,
		Password: password,
		TenantID: "", // We'll need to query this if needed
		UserID:   "", // We'll need to query this if needed
	}
}

// TestCreateRole_RBACFeatureFlag tests the RBAC feature flag functionality specifically
func TestCreateRole_RBACFeatureFlag(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB2(t, pool)

	// Create a new tenant via signup (all features disabled by default)
	signupEmail := "rbac_feature_test@example.com"
	signupPassword := "password123"
	tenantName := "RBAC Feature Test Tenant"
	userName := "RBAC Feature Test User"

	userCredentials := createTenantViaSignup(t, signupEmail, signupPassword, tenantName, userName)

	// Test 1: RBAC disabled - should get 403 Forbidden
	t.Run("RBAC disabled blocks access", func(t *testing.T) {
		// Login to get access token
		accessToken, _ := setup.LoginUserAndGetTokens(t, userCredentials.Email, userCredentials.Password)

		// Prepare create role request
		createRoleRequest := roles.CreateRoleRequestBody{
			Name:          "Test Role RBAC Disabled",
			Description:   "This should be blocked by RBAC feature flag",
			PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
		}

		jsonData, err := json.Marshal(createRoleRequest)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", setup.BaseURL+"/roles/create", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should be blocked by RBAC feature flag with 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "RBAC disabled should block access with 403")
	})

	// Test 2: Enable RBAC feature and verify access is allowed
	t.Run("RBAC enabled allows access", func(t *testing.T) {
		// First, enable RBAC feature for the tenant
		err := enableRBACForTenant(t, pool, userCredentials.Email)
		assert.NoError(t, err)

		// Login to get fresh access token
		accessToken, _ := setup.LoginUserAndGetTokens(t, userCredentials.Email, userCredentials.Password)

		// Prepare create role request
		createRoleRequest := roles.CreateRoleRequestBody{
			Name:          "Test Role RBAC Enabled",
			Description:   "This should succeed after enabling RBAC feature",
			PermissionIDs: []string{"3a52c807-ddcb-4044-8682-658e04800a8e"}, // users.view permission
		}

		jsonData, err := json.Marshal(createRoleRequest)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", setup.BaseURL+"/roles/create", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed now that RBAC is enabled
		assert.Equal(t, http.StatusOK, resp.StatusCode, "RBAC enabled should allow access")
	})
}

// enableRBACForTenant enables the RBAC feature for a tenant by email
func enableRBACForTenant(t *testing.T, pool *pgxpool.Pool, userEmail string) error {
	query := `
		UPDATE tenants 
		SET enterprise_features = '{"rbac": {"enabled": true}}'::jsonb
		WHERE id = (
			SELECT tenant_id 
			FROM users 
			WHERE email = $1
		)
	`

	_, err := pool.Exec(context.Background(), query, userEmail)
	if err != nil {
		return fmt.Errorf("failed to enable RBAC for tenant: %w", err)
	}

	return nil
}
