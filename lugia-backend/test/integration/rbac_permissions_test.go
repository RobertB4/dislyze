package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRBACPermissions_MeEndpoint(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	t.Run("RBAC Enabled - Enterprise Tenant", func(t *testing.T) {
		t.Run("User with 管理者", func(t *testing.T) {
			// enterprise_1 already has 管理者 role
			permissions := getMePermissions(t, "enterprise_1")
			
			// Should have all admin permissions
			assert.Contains(t, permissions, "tenant.edit")
			assert.Contains(t, permissions, "users.edit")
			assert.Contains(t, permissions, "roles.edit")
		})

		t.Run("User with 編集者", func(t *testing.T) {
			// enterprise_2 already has 編集者 role
			permissions := getMePermissions(t, "enterprise_2")
			
			// Should have no permissions (編集者 has no permissions assigned)
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})

		t.Run("User with 閲覧者", func(t *testing.T) {
			// enterprise_7 already has 閲覧者 role
			permissions := getMePermissions(t, "enterprise_7")
			
			// Should have no permissions (閲覧者 has no permissions assigned)
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})

		t.Run("User with ユーザー管理者 (custom)", func(t *testing.T) {
			// Assign only ユーザー管理者 (custom role) to enterprise_8
			userID := setup.TestUsersData["enterprise_8"].UserID
			tenantID := setup.TestUsersData["enterprise_8"].TenantID
			
			// Remove existing roles and assign only custom role
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{"dddddddd-dddd-dddd-dddd-dddddddddddd"}) // ユーザー管理者
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"cccccccc-cccc-cccc-cccc-cccccccccccc"}) // restore 閲覧者
			
			permissions := getMePermissions(t, "enterprise_8")
			
			// Should have only users.edit
			assert.Contains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "roles.edit")
		})

		t.Run("User with 管理者 + custom role", func(t *testing.T) {
			// Assign both 管理者 and ユーザー管理者 to enterprise_9
			userID := setup.TestUsersData["enterprise_9"].UserID
			tenantID := setup.TestUsersData["enterprise_9"].TenantID
			
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", // 管理者
				"dddddddd-dddd-dddd-dddd-dddddddddddd", // ユーザー管理者
			})
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"cccccccc-cccc-cccc-cccc-cccccccccccc"}) // restore 閲覧者
			
			permissions := getMePermissions(t, "enterprise_9")
			
			// Should have combined permissions (管理者 already includes users.edit)
			assert.Contains(t, permissions, "tenant.edit")
			assert.Contains(t, permissions, "users.edit")
			assert.Contains(t, permissions, "roles.edit")
		})

		t.Run("User with 編集者 + custom role", func(t *testing.T) {
			// Assign both 編集者 and ユーザー管理者 to enterprise_10
			userID := setup.TestUsersData["enterprise_10"].UserID
			tenantID := setup.TestUsersData["enterprise_10"].TenantID
			
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{
				"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", // 編集者
				"dddddddd-dddd-dddd-dddd-dddddddddddd", // ユーザー管理者
			})
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"cccccccc-cccc-cccc-cccc-cccccccccccc"}) // restore 閲覧者
			
			permissions := getMePermissions(t, "enterprise_10")
			
			// Should have users.edit from custom role (編集者 contributes zero permissions)
			assert.Contains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "roles.edit")
		})
	})

	t.Run("RBAC Disabled - SMB Tenant", func(t *testing.T) {
		t.Run("User with 管理者", func(t *testing.T) {
			// smb_1 already has 管理者 role
			permissions := getMePermissions(t, "smb_1")
			
			// Should have all admin permissions
			assert.Contains(t, permissions, "tenant.edit")
			assert.Contains(t, permissions, "users.edit")
			assert.Contains(t, permissions, "roles.edit")
		})

		t.Run("User with 編集者", func(t *testing.T) {
			// smb_2 already has 編集者 role
			permissions := getMePermissions(t, "smb_2")
			
			// Should have no permissions (編集者 has no permissions assigned)
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})

		t.Run("User with 閲覧者", func(t *testing.T) {
			// smb_7 already has 閲覧者 role
			permissions := getMePermissions(t, "smb_7")
			
			// Should have no permissions (閲覧者 has no permissions assigned)
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})

		t.Run("User with custom role ONLY", func(t *testing.T) {
			// Need to create a custom role for SMB tenant and assign it
			userID := setup.TestUsersData["smb_8"].UserID
			tenantID := setup.TestUsersData["smb_8"].TenantID
			
			// First create a custom role for SMB tenant (simulate ユーザー管理者)
			customRoleID := createCustomRole(t, pool, tenantID, "SMBユーザー管理者", false) // is_default = false
			assignPermissionToRole(t, pool, customRoleID, "db994eda-6ff7-4ae5-a675-3abe735ce9cc", tenantID) // users.edit
			
			// Remove existing roles and assign only custom role
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{customRoleID})
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"11111111-2222-3333-4444-555555555555"}) // restore 閲覧者
			defer deleteCustomRole(t, pool, customRoleID) // cleanup custom role
			
			permissions := getMePermissions(t, "smb_8")
			
			// Should get 閲覧者 fallback (which has no permissions)
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})

		t.Run("User with 管理者 + custom", func(t *testing.T) {
			// Assign both 管理者 and custom role to smb_9
			userID := setup.TestUsersData["smb_9"].UserID
			tenantID := setup.TestUsersData["smb_9"].TenantID
			
			// Create custom role with additional permission
			customRoleID := createCustomRole(t, pool, tenantID, "SMBカスタム", false)
			assignPermissionToRole(t, pool, customRoleID, "db994eda-6ff7-4ae5-a675-3abe735ce9cc", tenantID) // users.edit
			
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{
				"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", // 管理者
				customRoleID, // custom role
			})
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"11111111-2222-3333-4444-555555555555"}) // restore 閲覧者
			defer deleteCustomRole(t, pool, customRoleID) // cleanup custom role
			
			permissions := getMePermissions(t, "smb_9")
			
			// Should have only default role permissions (custom ignored)
			assert.Contains(t, permissions, "tenant.edit")
			assert.Contains(t, permissions, "users.edit")
			assert.Contains(t, permissions, "roles.edit")
		})

		t.Run("User with 編集者 + custom", func(t *testing.T) {
			// Assign both 編集者 and custom role to smb_10
			userID := setup.TestUsersData["smb_10"].UserID
			tenantID := setup.TestUsersData["smb_10"].TenantID
			
			// Create custom role with permission
			customRoleID := createCustomRole(t, pool, tenantID, "SMBエディタ", false)
			assignPermissionToRole(t, pool, customRoleID, "cccf277b-5fd5-4f1d-b763-ebf69973e5b7", tenantID) // roles.edit
			
			removeAllRolesFromUser(t, pool, userID, tenantID)
			assignRolesToUser(t, pool, userID, tenantID, []string{
				"ffffffff-ffff-ffff-ffff-ffffffffffff", // 編集者
				customRoleID, // custom role
			})
			defer removeAllRolesFromUser(t, pool, userID, tenantID) // cleanup
			defer assignRolesToUser(t, pool, userID, tenantID, []string{"11111111-2222-3333-4444-555555555555"}) // restore 閲覧者
			defer deleteCustomRole(t, pool, customRoleID) // cleanup custom role
			
			permissions := getMePermissions(t, "smb_10")
			
			// Should have only default role permissions (編集者 has zero, custom ignored)
			assert.NotContains(t, permissions, "roles.edit")
			assert.NotContains(t, permissions, "tenant.edit")
			assert.NotContains(t, permissions, "users.edit")
			assert.NotContains(t, permissions, "tenant.view")
			assert.NotContains(t, permissions, "users.view")
			assert.NotContains(t, permissions, "roles.view")
		})
	})
}

// Helper functions

func getMePermissions(t *testing.T, userKey string) []string {
	t.Helper()
	
	userDetails := setup.TestUsersData[userKey]
	accessToken, _ := setup.LoginUserAndGetTokens(t, userDetails.Email, userDetails.PlainTextPassword)
	
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})
	
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var meResponse users.MeResponse
	err = json.NewDecoder(resp.Body).Decode(&meResponse)
	require.NoError(t, err)
	
	return meResponse.Permissions
}

func assignRolesToUser(t *testing.T, pool *pgxpool.Pool, userID, tenantID string, roleIDs []string) {
	t.Helper()
	
	for _, roleID := range roleIDs {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)",
			userID, roleID, tenantID)
		require.NoError(t, err)
	}
}

func removeAllRolesFromUser(t *testing.T, pool *pgxpool.Pool, userID, tenantID string) {
	t.Helper()
	
	_, err := pool.Exec(context.Background(),
		"DELETE FROM user_roles WHERE user_id = $1 AND tenant_id = $2",
		userID, tenantID)
	require.NoError(t, err)
}

func createCustomRole(t *testing.T, pool *pgxpool.Pool, tenantID, roleName string, isDefault bool) string {
	t.Helper()
	
	var roleID string
	err := pool.QueryRow(context.Background(),
		"INSERT INTO roles (tenant_id, name, description, is_default) VALUES ($1, $2, $3, $4) RETURNING id",
		tenantID, roleName, "Test custom role", isDefault).Scan(&roleID)
	require.NoError(t, err)
	
	return roleID
}

func assignPermissionToRole(t *testing.T, pool *pgxpool.Pool, roleID, permissionID, tenantID string) {
	t.Helper()
	
	_, err := pool.Exec(context.Background(),
		"INSERT INTO role_permissions (role_id, permission_id, tenant_id) VALUES ($1, $2, $3)",
		roleID, permissionID, tenantID)
	require.NoError(t, err)
}

func deleteCustomRole(t *testing.T, pool *pgxpool.Pool, roleID string) {
	t.Helper()
	
	// Delete role permissions first
	_, err := pool.Exec(context.Background(),
		"DELETE FROM role_permissions WHERE role_id = $1", roleID)
	require.NoError(t, err)
	
	// Delete user role assignments
	_, err = pool.Exec(context.Background(),
		"DELETE FROM user_roles WHERE role_id = $1", roleID)
	require.NoError(t, err)
	
	// Delete the role
	_, err = pool.Exec(context.Background(),
		"DELETE FROM roles WHERE id = $1", roleID)
	require.NoError(t, err)
}