package audit_logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"lugia/features/audit_logs"
	"lugia/features/roles"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getAuditLogs fetches audit logs with optional query params
func getAuditLogs(t *testing.T, accessToken string, queryParams string) (*audit_logs.GetAuditLogsResponse, int) {
	t.Helper()
	client := &http.Client{}

	url := fmt.Sprintf("%s/audit-logs?%s", setup.BaseURL, queryParams)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode
	}

	var result audit_logs.GetAuditLogsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return &result, resp.StatusCode
}

func TestAuditLogs_SeedDataVisible(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Enterprise admin (enterprise_1) has audit_log.view permission
	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["enterprise_1"].Email,
		setup.TestUsersData["enterprise_1"].PlainTextPassword)

	result, status := getAuditLogs(t, accessToken, "limit=100")
	assert.Equal(t, http.StatusOK, status)
	assert.NotNil(t, result)

	// Seed data includes sample audit log entries + login events from LoginUserAndGetTokens
	assert.Greater(t, len(result.AuditLogs), 0, "Should have seed audit log entries")
}

func TestAuditLogs_EnterpriseGating(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// SMB tenant does NOT have audit_log enabled
	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["smb_1"].Email,
		setup.TestUsersData["smb_1"].PlainTextPassword)

	_, status := getAuditLogs(t, accessToken, "")
	assert.Equal(t, http.StatusForbidden, status, "SMB tenant without audit_log feature should get 403")
}

func TestAuditLogs_Unauthenticated(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/audit-logs", setup.BaseURL), nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuditLogs_TenantIsolation(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Both tenants have audit_log enabled (enterprise and internal via seed data).
	// Login as both to generate audit log entries for each tenant.
	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	internalAdmin := setup.TestUsersData["internal_1"]

	// Login generates audit log entries for each tenant
	enterpriseToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)
	internalToken, _ := setup.LoginUserAndGetTokens(t, internalAdmin.Email, internalAdmin.PlainTextPassword)

	// Verify both tenants have entries in the DB
	var enterpriseCount, internalCount int
	err := pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1",
		setup.TestTenantsData["enterprise"].ID).Scan(&enterpriseCount)
	require.NoError(t, err)
	require.Greater(t, enterpriseCount, 0, "Enterprise tenant should have audit log entries in DB")

	err = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1",
		setup.TestTenantsData["internal"].ID).Scan(&internalCount)
	require.NoError(t, err)
	require.Greater(t, internalCount, 0, "Internal tenant should have audit log entries in DB")

	// Enterprise user should only see enterprise entries
	enterpriseResult, status := getAuditLogs(t, enterpriseToken, "limit=100")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(enterpriseResult.AuditLogs), 0)
	for _, entry := range enterpriseResult.AuditLogs {
		assert.NotEqual(t, internalAdmin.UserID, entry.ActorID,
			"Enterprise user should not see internal tenant's audit logs")
	}

	// Internal user should only see internal entries
	internalResult, status := getAuditLogs(t, internalToken, "limit=100")
	assert.Equal(t, http.StatusOK, status)
	require.Greater(t, len(internalResult.AuditLogs), 0)
	for _, entry := range internalResult.AuditLogs {
		assert.NotEqual(t, enterpriseAdmin.UserID, entry.ActorID,
			"Internal user should not see enterprise tenant's audit logs")
	}
}

func TestAuditLogs_FilterByAction(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["enterprise_1"].Email,
		setup.TestUsersData["enterprise_1"].PlainTextPassword)

	// Filter for login events only
	result, status := getAuditLogs(t, accessToken, "action=login&limit=100")
	assert.Equal(t, http.StatusOK, status)

	for _, entry := range result.AuditLogs {
		assert.Equal(t, "login", entry.Action, "All entries should be login actions")
	}
}

func TestAuditLogs_FilterByResourceType(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["enterprise_1"].Email,
		setup.TestUsersData["enterprise_1"].PlainTextPassword)

	// Filter for role resource type
	result, status := getAuditLogs(t, accessToken, "resource_type=role&limit=100")
	assert.Equal(t, http.StatusOK, status)

	for _, entry := range result.AuditLogs {
		assert.Equal(t, "role", entry.ResourceType, "All entries should be role resource type")
	}
}

func TestAuditLogs_FilterByOutcome(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["enterprise_1"].Email,
		setup.TestUsersData["enterprise_1"].PlainTextPassword)

	result, status := getAuditLogs(t, accessToken, "outcome=success&limit=100")
	assert.Equal(t, http.StatusOK, status)

	for _, entry := range result.AuditLogs {
		assert.Equal(t, "success", entry.Outcome, "All entries should have success outcome")
	}
}

func TestAuditLogs_Pagination(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	accessToken, _ := setup.LoginUserAndGetTokens(t,
		setup.TestUsersData["enterprise_1"].Email,
		setup.TestUsersData["enterprise_1"].PlainTextPassword)

	// Get page 1 with limit 2
	result, status := getAuditLogs(t, accessToken, "page=1&limit=2")
	assert.Equal(t, http.StatusOK, status)
	assert.LessOrEqual(t, len(result.AuditLogs), 2, "Should return at most 2 entries")
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 2, result.Pagination.Limit)

	if result.Pagination.HasNext {
		// Get page 2
		result2, status2 := getAuditLogs(t, accessToken, "page=2&limit=2")
		assert.Equal(t, http.StatusOK, status2)
		assert.LessOrEqual(t, len(result2.AuditLogs), 2)
		assert.Equal(t, 2, result2.Pagination.Page)

		// Pages should have different entries
		if len(result.AuditLogs) > 0 && len(result2.AuditLogs) > 0 {
			assert.NotEqual(t, result.AuditLogs[0].ID, result2.AuditLogs[0].ID,
				"Different pages should have different entries")
		}
	}
}

func TestAuditLogs_RoleCreatedEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Clear existing audit logs to isolate this test
	_, err := pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Create a role
	createRoleRequest := roles.CreateRoleRequestBody{
		Name:          "Audit Test Role",
		Description:   "Role created to test audit logging",
		PermissionIDs: []string{setup.TestPermissionsData["users_view"].ID},
	}
	jsonData, err := json.Marshal(createRoleRequest)
	require.NoError(t, err)

	client := &http.Client{}
	req, err := http.NewRequest("POST", setup.BaseURL+"/roles/create", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})

	resp, err := client.Do(req)
	require.NoError(t, err)
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
	err = json.Unmarshal(entry.Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Audit Test Role", metadata["role_name"])
}

func TestAuditLogs_LoginEvent(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// Clear existing audit logs
	_, err := pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)

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

	// Clear existing audit logs
	_, err := pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)

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

	// Clear existing audit logs
	_, err := pool.Exec(context.Background(), "DELETE FROM audit_logs")
	require.NoError(t, err)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Access users list
	client := &http.Client{}
	req, err := http.NewRequest("GET", setup.BaseURL+"/users", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})

	resp, err := client.Do(req)
	require.NoError(t, err)
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

func TestAuditLogs_PermissionDeniedNotLogged_ForNonAuditTenant(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	// SMB tenant doesn't have audit_log enabled, so access failures should NOT be logged
	smbUser := setup.TestUsersData["smb_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, smbUser.Email, smbUser.PlainTextPassword)

	// Try to access roles (SMB doesn't have RBAC), should get 403
	client := &http.Client{}
	req, err := http.NewRequest("GET", setup.BaseURL+"/roles", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken})

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// Verify no audit log entry was created (SMB doesn't have audit_log feature)
	var count int
	err = pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND resource_type = 'access'",
		setup.TestTenantsData["smb"].ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "SMB tenant should not have access denial audit logs")
}

func TestAuditLogs_FilterByActorID(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	setup.ResetAndSeedDB(t, pool)

	enterpriseAdmin := setup.TestUsersData["enterprise_1"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseAdmin.Email, enterpriseAdmin.PlainTextPassword)

	// Filter by enterprise_1's actor ID
	result, status := getAuditLogs(t, accessToken,
		fmt.Sprintf("actor_id=%s&limit=100", enterpriseAdmin.UserID))
	assert.Equal(t, http.StatusOK, status)

	for _, entry := range result.AuditLogs {
		assert.Equal(t, enterpriseAdmin.UserID, entry.ActorID,
			"All entries should be from the specified actor")
	}
}
