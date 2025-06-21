package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"lugia/features/ip_whitelist"
	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a role with specific IP whitelist permissions
func createIPWhitelistRole(t *testing.T, pool *pgxpool.Pool, tenantID string, permissionIDs []string) string {
	ctx := context.Background()

	// Insert role into database
	var roleID string
	err := pool.QueryRow(ctx,
		`INSERT INTO roles (tenant_id, name, description, is_default) 
		 VALUES ($1, $2, $3, false) 
		 RETURNING id`,
		tenantID, "IPホワイトリストテスト役割", "テスト用役割").Scan(&roleID)
	assert.NoError(t, err)

	// Insert role permissions
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

// Helper function to update tenant enterprise features
func updateTenantEnterpriseFeatures(t *testing.T, pool *pgxpool.Pool, tenantID string, features map[string]interface{}) {
	ctx := context.Background()
	featuresJSON, err := json.Marshal(features)
	assert.NoError(t, err)

	_, err = pool.Exec(ctx,
		`UPDATE tenants SET enterprise_features = $1 WHERE id = $2`,
		featuresJSON, tenantID)
	assert.NoError(t, err)
}

// Helper function to insert IP whitelist rule
func insertIPWhitelistRule(t *testing.T, pool *pgxpool.Pool, tenantID, ipAddress, label, createdBy string) {
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, label, created_by) 
		 VALUES ($1, $2, $3, $4)`,
		tenantID, ipAddress, label, createdBy)
	assert.NoError(t, err)
}

// Helper function to find user credentials by key
func findUserCredentials(userKey string) (email string, password string) {
	userData, exists := setup.TestUsersData[userKey]
	if !exists {
		return "", ""
	}
	return userData.Email, userData.PlainTextPassword
}

func TestGetIPWhitelistIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)


	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, pool *pgxpool.Pool) (string, string) // Returns (userKey, clientIP)
		expectedStatus int
		validateFunc   func(t *testing.T, response []ip_whitelist.IPWhitelistRule)
	}{
		{
			name: "test_unauthenticated_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				return "", "192.168.1.100" // No user login
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_no_ip_whitelist_permission_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Use enterprise_2 who has editor role but no IP whitelist permissions
				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_ip_whitelist_view_permission_succeeds",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=false
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return empty list since no IP rules are configured
				assert.Equal(t, 0, len(response))
			},
		},
		{
			name: "test_ip_whitelist_edit_permission_succeeds",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.edit permission for enterprise_3
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_edit"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=false
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_3", "192.168.1.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return empty list since no IP rules are configured
				// Edit permission inherits view permission
				assert.Equal(t, 0, len(response))
			},
		},
		{
			name: "test_feature_disabled_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=false (feature disabled)
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     false, // Feature disabled - should block access
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_feature_enabled_but_inactive_succeeds",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=false (middleware skips enforcement)
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false, // Not active - middleware skips IP checks
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return empty list since no IP rules are configured
				assert.Equal(t, 0, len(response))
			},
		},
		{
			name: "test_feature_active_empty_whitelist_blocks_all",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true (middleware enforces IP restrictions)
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true, // Active - middleware enforces IP checks
						"allow_internal_admin_bypass": false,
					},
				})

				// Don't insert any IP rules - empty whitelist should block all IPs

				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusForbidden, // IP whitelist middleware blocks access
		},
		{
			name: "test_active_feature_blocks_non_whitelisted_ipv4",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IP rule for 10.0.0.0/8 range
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 192.168.1.100 is NOT in 10.0.0.0/8 range - should be blocked
				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusForbidden, // IP whitelist middleware blocks non-whitelisted IP
		},
		{
			name: "test_active_feature_allows_whitelisted_ipv4",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add multiple IP rules including one that matches client IP
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Office Network", setup.TestUsersData["enterprise_1"].UserID)
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.100/32", "VPN Gateway", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 192.168.1.50 is in 192.168.1.0/24 range - should be allowed
				return "enterprise_2", "192.168.1.50"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return the 2 IP rules we inserted
				assert.Equal(t, 2, len(response))

				// Validate response structure
				for _, rule := range response {
					assert.NotEmpty(t, rule.ID)
					assert.NotEmpty(t, rule.IPAddress)
					assert.NotEmpty(t, rule.CreatedBy)
					assert.NotZero(t, rule.CreatedAt)
					// rule.Label can be nil or have value
				}

				// Validate specific IP addresses are present
				ips := make([]string, len(response))
				for i, rule := range response {
					ips[i] = rule.IPAddress
				}
				assert.Contains(t, ips, "192.168.1.0/24")
				assert.Contains(t, ips, "10.0.0.100/32")
			},
		},
		{
			name: "test_ipv4_exact_match_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add exact IP match rule
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.100/32", "Exact IP", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 10.0.0.100 exactly matches the rule - should be allowed
				return "enterprise_2", "10.0.0.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "10.0.0.100/32", response[0].IPAddress)
			},
		},
		{
			name: "test_ipv4_cidr_range_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add CIDR range rule
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "172.16.0.0/12", "Private Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 172.20.5.10 is within 172.16.0.0/12 range - should be allowed
				return "enterprise_2", "172.20.5.10"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "172.16.0.0/12", response[0].IPAddress)
			},
		},
		{
			name: "test_multiple_ipv4_rules_any_match_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add multiple IP rules - client should match the third one
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Office Network", setup.TestUsersData["enterprise_1"].UserID)
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "172.16.0.100/32", "Specific Server", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 172.16.0.100 matches the third rule exactly - should be allowed
				return "enterprise_2", "172.16.0.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return all 3 IP rules
				assert.Equal(t, 3, len(response))

				// Validate specific IP addresses are present
				ips := make([]string, len(response))
				for i, rule := range response {
					ips[i] = rule.IPAddress
				}
				assert.Contains(t, ips, "10.0.0.0/8")
				assert.Contains(t, ips, "192.168.1.0/24")
				assert.Contains(t, ips, "172.16.0.100/32")
			},
		},
		{
			name: "test_ipv6_exact_match_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IPv6 exact match rule
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "2001:db8::1/128", "IPv6 Server", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 2001:db8::1 exactly matches the rule - should be allowed
				return "enterprise_2", "2001:db8::1"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "2001:db8::1/128", response[0].IPAddress)
			},
		},
		{
			name: "test_ipv6_cidr_range_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IPv6 CIDR range rule
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "2001:db8::/64", "IPv6 Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 2001:db8::50 is within 2001:db8::/64 range - should be allowed
				return "enterprise_2", "2001:db8::50"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "2001:db8::/64", response[0].IPAddress)
			},
		},
		{
			name: "test_ipv6_blocks_non_whitelisted",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IPv6 CIDR range rule
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "2001:db8::/64", "IPv6 Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 2001:db9::1 is NOT in 2001:db8::/64 range - should be blocked
				return "enterprise_2", "2001:db9::1"
			},
			expectedStatus: http.StatusForbidden, // IP whitelist middleware blocks non-whitelisted IPv6
		},
		{
			name: "test_internal_user_bypass_enabled_allows_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for internal_user_enterprise
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["internal_user_enterprise"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true, bypass=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": true, // Allow internal user bypass
					},
				})

				// Add IP rule for 10.0.0.0/8 range
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 192.168.1.100 is NOT whitelisted, but internal user should bypass
				return "internal_user_enterprise", "192.168.1.100"
			},
			expectedStatus: http.StatusOK, // Internal user bypasses IP restrictions
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "10.0.0.0/8", response[0].IPAddress)
			},
		},
		{
			name: "test_internal_user_bypass_disabled_blocks_access",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for internal_user_enterprise
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["internal_user_enterprise"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true, bypass=false
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false, // Disable internal user bypass
					},
				})

				// Add IP rule for 10.0.0.0/8 range
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)

				// Client IP 192.168.1.100 is NOT whitelisted, and bypass disabled - should be blocked
				return "internal_user_enterprise", "192.168.1.100"
			},
			expectedStatus: http.StatusForbidden, // Even internal users blocked when bypass disabled
		},
		{
			name: "test_tenant_isolation_smb_cannot_see_enterprise_rules",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// First, add IP rules to Enterprise tenant
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "10.0.0.0/8", "Enterprise Network", setup.TestUsersData["enterprise_1"].UserID)
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Enterprise Office", setup.TestUsersData["enterprise_1"].UserID)

				// Create role with ip_whitelist.view permission for SMB user
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["smb"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["smb_1"].UserID, roleID, setup.TestTenantsData["smb"].ID)

				// Set enterprise features for SMB tenant
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["smb"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "smb_1", "192.168.1.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// SMB user should see no Enterprise rules (empty response)
				assert.Equal(t, 0, len(response))
			},
		},
		{
			name: "test_tenant_isolation_enterprise_cannot_see_smb_rules",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// First, add IP rules to SMB tenant
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["smb"].ID, "172.16.0.0/12", "SMB Network", setup.TestUsersData["smb_1"].UserID)
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["smb"].ID, "203.0.113.0/24", "SMB External", setup.TestUsersData["smb_1"].UserID)

				// Create role with ip_whitelist.view permission for Enterprise user
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features for Enterprise tenant
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      false,
						"allow_internal_admin_bypass": false,
					},
				})

				return "enterprise_2", "192.168.1.100"
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Enterprise user should see no SMB rules (empty response)
				assert.Equal(t, 0, len(response))
			},
		},
		{
			name: "test_x_forwarded_for_header_extraction",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IP rule for 172.16.0.0/12 range
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "172.16.0.0/12", "Internal Network", setup.TestUsersData["enterprise_1"].UserID)

				// Return special marker for X-Forwarded-For test - rightmost IP should be extracted (GCP Load Balancer behavior)
				return "enterprise_2", "203.0.113.50, 192.168.1.1, 172.16.0.1"
			},
			expectedStatus: http.StatusOK, // Rightmost IP (172.16.0.1) matches 172.16.0.0/12
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "172.16.0.0/12", response[0].IPAddress)
			},
		},
		{
			name: "test_x_real_ip_header_extraction",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string) {
				// Create role with ip_whitelist.view permission for enterprise_2
				roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
					setup.TestPermissionsData["ip_whitelist_view"].ID,
				})
				assignRoleToUser(t, pool, setup.TestUsersData["enterprise_2"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

				// Set enterprise features: enabled=true, active=true
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true,
						"allow_internal_admin_bypass": false,
					},
				})

				// Add IP rule for 192.168.1.0/24 range
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Office Network", setup.TestUsersData["enterprise_1"].UserID)

				// Return special marker for X-Real-IP test
				return "enterprise_2", "X-Real-IP:192.168.1.75"
			},
			expectedStatus: http.StatusOK, // X-Real-IP (192.168.1.75) matches 192.168.1.0/24
			validateFunc: func(t *testing.T, response []ip_whitelist.IPWhitelistRule) {
				// Should return 1 IP rule
				assert.Equal(t, 1, len(response))
				assert.Equal(t, "192.168.1.0/24", response[0].IPAddress)
			},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset database state before each test case
			setup.ResetAndSeedDB(t, pool)

			userKey, clientIP := tt.setupFunc(t, pool)

			// Create request
			reqURL := fmt.Sprintf("%s/ip-whitelist", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			// Set client IP header(s) based on test case
			if strings.Contains(clientIP, "X-Real-IP:") {
				// Special case for X-Real-IP header test
				realIP := strings.TrimPrefix(clientIP, "X-Real-IP:")
				req.Header.Set("X-Real-IP", realIP)
			} else if strings.Contains(clientIP, ",") {
				// Special case for X-Forwarded-For header test with multiple IPs
				req.Header.Set("X-Forwarded-For", clientIP)
			} else {
				// Normal case - set X-Forwarded-For header
				req.Header.Set("X-Forwarded-For", clientIP)
			}

			// Login if user specified
			if userKey != "" {
				email, password := findUserCredentials(userKey)
				require.NotEmpty(t, email, "User key not found: %s", userKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			// Make request
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				// Add debug information for failed tests
				body := make([]byte, 1024)
				n, _ := resp.Body.Read(body)
				t.Logf("Expected status %d, got %d. Response body: %s", tt.expectedStatus, resp.StatusCode, string(body[:n]))
			}
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// If success, validate response
			if tt.expectedStatus == http.StatusOK && tt.validateFunc != nil {
				var response []ip_whitelist.IPWhitelistRule
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err, "Should be able to decode JSON response")

				tt.validateFunc(t, response)
			}
		})
	}
}
