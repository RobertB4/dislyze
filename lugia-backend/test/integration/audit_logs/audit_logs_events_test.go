// Feature doc: docs/features/audit-logging.md
package audit_logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/roles"
	"lugia/features/users"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateTenantEnterpriseFeatures updates the enterprise_features JSON for a tenant.
func updateTenantEnterpriseFeatures(t *testing.T, pool *pgxpool.Pool, tenantID string, features map[string]interface{}) {
	t.Helper()
	featuresJSON, err := json.Marshal(features)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`UPDATE tenants SET enterprise_features = $1 WHERE id = $2`,
		featuresJSON, tenantID)
	require.NoError(t, err)
}

// doAuthenticatedRequest makes an HTTP request with auth cookie and returns the response.
func doAuthenticatedRequest(t *testing.T, method, url string, body interface{}, accessToken string) *http.Response {
	t.Helper()
	client := &http.Client{}

	var req *http.Request
	var err error
	if body != nil {
		jsonData, marshalErr := json.Marshal(body)
		require.NoError(t, marshalErr)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// clearAuditLogs deletes all audit log entries to isolate test assertions.
func clearAuditLogs(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)
}

func TestAuditLogs_RoleCreatedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Create a role
	createRoleRequest := roles.CreateRoleRequestBody{
		Name:          "Audit Test Role",
		Description:   "Role created to test audit logging",
		PermissionIDs: []string{setup.TestPermissionsData["users_view"].ID},
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/roles/create", createRoleRequest, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "Role creation should succeed")

	// Verify audit log entry was created
	result, status := getAuditLogs(t, accessToken, "resource_type=role&action=created&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have role.created audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "role", entry.ResourceType)
	assert.Equal(t, "created", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)
	assert.Equal(t, enterpriseAdmin.Name, entry.ActorName)
	assert.Equal(t, enterpriseAdmin.Email, entry.ActorEmail)
	assert.NotNil(t, entry.ResourceID, "Role creation should include resource_id")

	// Verify metadata contains role_name
	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Audit Test Role", metadata["role_name"])
}

func TestAuditLogs_LoginEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]

	// Perform a login (which generates an audit event)
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Check for login audit entry
	result, status := getAuditLogs(t, accessToken, "resource_type=auth&action=login&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have auth.login audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "auth", entry.ResourceType)
	assert.Equal(t, "login", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)
}

func TestAuditLogs_LoginFailureEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]

	// Attempt login with wrong password
	resp := setup.AttemptLogin(t, enterpriseAdmin.Email, "wrong_password")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Login with correct password to access audit logs
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Check for failed login audit entry
	result, status := getAuditLogs(t, accessToken, "resource_type=auth&action=login&outcome=failure&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have auth.login failure audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "auth", entry.ResourceType)
	assert.Equal(t, "login", entry.Action)
	assert.Equal(t, "failure", entry.Outcome)
}

func TestAuditLogs_UserListViewedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Access users list
	resp := doAuthenticatedRequest(t, "GET", setup.BaseURL+"/users", nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify GDPR read access audit entry
	result, status := getAuditLogs(t, accessToken, "resource_type=user&action=list_viewed&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have user.list_viewed audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "user", entry.ResourceType)
	assert.Equal(t, "list_viewed", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
}

func TestAuditLogs_LogoutEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Perform logout
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/auth/logout", nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Re-login to access audit logs
	accessToken, _ = setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	result, status := getAuditLogs(t, accessToken, "resource_type=auth&action=logout&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have auth.logout audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "auth", entry.ResourceType)
	assert.Equal(t, "logout", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)
}

func TestAuditLogs_PasswordChangedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Change password
	changePasswordReq := users.ChangePasswordRequestBody{
		CurrentPassword:    enterpriseAdmin.PlainTextPassword,
		NewPassword:        "new_password_123",
		NewPasswordConfirm: "new_password_123",
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/me/change-password", changePasswordReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Re-login with new password to access audit logs
	accessToken, _ = setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, "new_password_123")

	result, status := getAuditLogs(t, accessToken, "resource_type=auth&action=password_changed&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have auth.password_changed audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "auth", entry.ResourceType)
	assert.Equal(t, "password_changed", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)
}

func TestAuditLogs_RoleUpdatedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Create a non-default role first (default roles can't be updated)
	createReq := roles.CreateRoleRequestBody{
		Name:          "Updatable Role",
		Description:   "Will be updated",
		PermissionIDs: []string{setup.TestPermissionsData["users_view"].ID},
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/roles/create", createReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Find the created role ID from audit logs
	result, _ := getAuditLogs(t, accessToken, "resource_type=role&action=created&limit=1")
	require.Greater(t, len(result.AuditLogs), 0)
	roleID := *result.AuditLogs[0].ResourceID

	// Clear and update the role
	clearAuditLogs(t, pool)
	// Re-login since clearAuditLogs doesn't affect tokens
	accessToken, _ = setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	updateReq := roles.UpdateRoleRequestBody{
		Name:          "Updated Role Name",
		Description:   "Updated description",
		PermissionIDs: []string{setup.TestPermissionsData["users_view"].ID, setup.TestPermissionsData["users_edit"].ID},
	}
	resp2 := doAuthenticatedRequest(t, "POST", fmt.Sprintf("%s/roles/%s/update", setup.BaseURL, roleID), updateReq, accessToken)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp2.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=role&action=updated&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have role.updated audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "role", entry.ResourceType)
	assert.Equal(t, "updated", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, roleID, *entry.ResourceID)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Updated Role Name", metadata["role_name"])
}

func TestAuditLogs_RoleDeletedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Create a non-default role to delete
	createReq := roles.CreateRoleRequestBody{
		Name:          "Deletable Role",
		Description:   "Will be deleted",
		PermissionIDs: []string{setup.TestPermissionsData["users_view"].ID},
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/roles/create", createReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Find the created role ID
	result, _ := getAuditLogs(t, accessToken, "resource_type=role&action=created&limit=1")
	require.Greater(t, len(result.AuditLogs), 0)
	roleID := *result.AuditLogs[0].ResourceID

	// Clear and delete the role
	clearAuditLogs(t, pool)
	accessToken, _ = setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	resp2 := doAuthenticatedRequest(t, "POST", fmt.Sprintf("%s/roles/%s/delete", setup.BaseURL, roleID), nil, accessToken)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp2.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=role&action=deleted&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have role.deleted audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "role", entry.ResourceType)
	assert.Equal(t, "deleted", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, roleID, *entry.ResourceID)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Deletable Role", metadata["role_name"])
}

func TestAuditLogs_UserDeletedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Delete enterprise_2 (a different user, since self-deletion returns 409)
	targetUser := setup.TestUsersData["enterprise_2"]
	resp := doAuthenticatedRequest(t, "POST", fmt.Sprintf("%s/users/%s/delete", setup.BaseURL, targetUser.UserID), nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=user&action=deleted&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have user.deleted audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "user", entry.ResourceType)
	assert.Equal(t, "deleted", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)
	assert.Equal(t, targetUser.UserID, *entry.ResourceID)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, targetUser.Email, metadata["deleted_user_email"])
}

func TestAuditLogs_TenantNameChangedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	changeNameReq := users.ChangeTenantNameRequestBody{
		Name: "新しいテナント名",
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/tenant/change-name", changeNameReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=tenant&action=name_changed&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have tenant.name_changed audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "tenant", entry.ResourceType)
	assert.Equal(t, "name_changed", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "新しいテナント名", metadata["new_name"])
}

func TestAuditLogs_UserRolesUpdatedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Update enterprise_3's roles
	targetUser := setup.TestUsersData["enterprise_3"]
	updateRolesReq := users.UpdateUserRolesRequestBody{
		RoleIDs: []string{
			setup.TestRolesData["enterprise_admin"].ID,
			setup.TestRolesData["enterprise_editor"].ID,
		},
	}
	resp := doAuthenticatedRequest(t, "POST", fmt.Sprintf("%s/users/%s/roles", setup.BaseURL, targetUser.UserID), updateRolesReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=user&action=roles_updated&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have user.roles_updated audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "user", entry.ResourceType)
	assert.Equal(t, "roles_updated", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
	assert.Equal(t, targetUser.UserID, *entry.ResourceID)
}

func TestAuditLogs_IPWhitelistAddIPEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Enable ip_whitelist feature for enterprise tenant
	updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
		"rbac":         map[string]interface{}{"enabled": true},
		"audit_log":    map[string]interface{}{"enabled": true},
		"ip_whitelist": map[string]interface{}{"enabled": true, "active": false},
	})
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	label := "Test IP"
	addIPReq := map[string]interface{}{
		"ip_address": "192.168.1.0/24",
		"label":      &label,
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/ip-whitelist/create", addIPReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=ip_whitelist&action=ip_added&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have ip_whitelist.ip_added audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "ip_whitelist", entry.ResourceType)
	assert.Equal(t, "ip_added", entry.Action)
	assert.Equal(t, "success", entry.Outcome)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.0/24", metadata["ip_address"])
}

func TestAuditLogs_IPWhitelistActivateEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Enable ip_whitelist feature and add an IP so activation can succeed
	updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
		"rbac":         map[string]interface{}{"enabled": true},
		"audit_log":    map[string]interface{}{"enabled": true},
		"ip_whitelist": map[string]interface{}{"enabled": true, "active": false},
	})

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]

	// Add wide IP ranges (IPv4 + IPv6) so the test client isn't blocked after activation
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, created_by) VALUES ($1, '0.0.0.0/0', $2), ($1, '::/0', $2)`,
		setup.TestTenantsData["enterprise"].ID, enterpriseAdmin.UserID)
	require.NoError(t, err)

	clearAuditLogs(t, pool)
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Activate whitelist with force=true to skip IP check
	activateReq := map[string]interface{}{
		"force": true,
	}
	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/ip-whitelist/activate", activateReq, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=ip_whitelist&action=activated&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have ip_whitelist.activated audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "ip_whitelist", entry.ResourceType)
	assert.Equal(t, "activated", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
}

func TestAuditLogs_IPWhitelistDeactivateEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Enable ip_whitelist feature (not active — avoids IP whitelist middleware blocking test requests)
	updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
		"rbac":         map[string]interface{}{"enabled": true},
		"audit_log":    map[string]interface{}{"enabled": true},
		"ip_whitelist": map[string]interface{}{"enabled": true, "active": false},
	})

	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	resp := doAuthenticatedRequest(t, "POST", setup.BaseURL+"/ip-whitelist/deactivate", nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=ip_whitelist&action=deactivated&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have ip_whitelist.deactivated audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "ip_whitelist", entry.ResourceType)
	assert.Equal(t, "deactivated", entry.Action)
	assert.Equal(t, "success", entry.Outcome)
}

func TestAuditLogs_PermissionDeniedEvent_ForEnterpriseTenant(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)
	clearAuditLogs(t, pool)

	// enterprise_3 has editor role which doesn't have users_edit permission
	// Try to delete a user — should get 403 and produce access.permission_denied audit entry
	enterpriseEditor := setup.TestUsersData["enterprise_3"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseEditor.Email, enterpriseEditor.PlainTextPassword)

	targetUser := setup.TestUsersData["enterprise_2"]
	resp := doAuthenticatedRequest(t, "POST", fmt.Sprintf("%s/users/%s/delete", setup.BaseURL, targetUser.UserID), nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Login as admin to view audit logs (editor may not have audit_log.view permission)
	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	adminToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	result, status := getAuditLogs(t, adminToken, "resource_type=access&action=permission_denied&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have access.permission_denied audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "access", entry.ResourceType)
	assert.Equal(t, "permission_denied", entry.Action)
	assert.Equal(t, "failure", entry.Outcome)
	assert.Equal(t, enterpriseEditor.UserID, entry.ActorID)
}

func TestAuditLogs_FeatureGateBlockedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Enable audit_log but disable ip_whitelist for enterprise tenant
	updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
		"rbac":         map[string]interface{}{"enabled": true},
		"audit_log":    map[string]interface{}{"enabled": true},
		"ip_whitelist": map[string]interface{}{"enabled": false},
	})
	clearAuditLogs(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Try to access ip-whitelist — should get 403 (feature gate blocked)
	resp := doAuthenticatedRequest(t, "GET", setup.BaseURL+"/ip-whitelist", nil, accessToken)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	result, status := getAuditLogs(t, accessToken, "resource_type=access&action=feature_gate_blocked&limit=10")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(result.AuditLogs), 0, "Should have access.feature_gate_blocked audit entry")

	entry := result.AuditLogs[0]
	assert.Equal(t, "access", entry.ResourceType)
	assert.Equal(t, "feature_gate_blocked", entry.Action)
	assert.Equal(t, "failure", entry.Outcome)

	var metadata map[string]string
	err := json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "ip_whitelist", metadata["feature"])
}

func TestAuditLogs_DateRangeFilter(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Filter with a future date range — should return no entries
	result, status := getAuditLogs(t, accessToken, "from_date=2099-01-01T00:00:00Z&to_date=2099-12-31T23:59:59Z&limit=100")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, len(result.AuditLogs), "Future date range should return no entries")

	// Filter with a past-to-now range — should return entries
	result, status = getAuditLogs(t, accessToken, "from_date=2020-01-01T00:00:00Z&limit=100")
	assert.Equal(t, http.StatusOK, status)
	assert.Greater(t, len(result.AuditLogs), 0, "Past date range should return entries")
}
