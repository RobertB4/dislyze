package ip_whitelist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// Helper function to insert IP whitelist rule and return the ID
func insertIPWhitelistRuleAndReturnID(t *testing.T, pool *pgxpool.Pool, tenantID, ipAddress, label, createdBy string) string {
	ctx := context.Background()
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO tenant_ip_whitelist (tenant_id, ip_address, label, created_by) 
		 VALUES ($1, $2, $3, $4) 
		 RETURNING id`,
		tenantID, ipAddress, label, createdBy).Scan(&id)
	assert.NoError(t, err)
	return id
}

func TestUpdateIPLabelIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	t.Run("test_unauthenticated_returns_401", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request (using dummy ID)
		reqURL := fmt.Sprintf("%s/ip-whitelist/11111111-1111-1111-1111-111111111111/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Don't set any authentication - should return 401

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("test_no_ip_whitelist_permission_returns_403", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Use enterprise_2 who has editor role but no IP whitelist permissions
		email, password := findUserCredentials("enterprise_2")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request (using dummy ID)
		reqURL := fmt.Sprintf("%s/ip-whitelist/11111111-1111-1111-1111-111111111111/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 403 because user lacks IP whitelist permissions
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("test_feature_disabled_returns_403", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Set enterprise features: ip_whitelist.enabled = false (feature disabled)
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

		// Use enterprise_1 who already has IP whitelist permissions
		email, password := findUserCredentials("enterprise_1")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request (using dummy ID)
		reqURL := fmt.Sprintf("%s/ip-whitelist/11111111-1111-1111-1111-111111111111/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 403 because feature is disabled
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("test_invalid_json_returns_400", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Set enterprise features: enabled=true
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

		// Use enterprise_1 who already has IP whitelist permissions
		email, password := findUserCredentials("enterprise_1")

		// Create malformed JSON body
		malformedJSON := `{"label": "Updated Label"` // Missing closing brace

		// Create request (using dummy ID)
		reqURL := fmt.Sprintf("%s/ip-whitelist/11111111-1111-1111-1111-111111111111/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer([]byte(malformedJSON)))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 400 because JSON is malformed
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_invalid_uuid_returns_400", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Set enterprise features: enabled=true
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

		// Use enterprise_1 who already has IP whitelist permissions
		email, password := findUserCredentials("enterprise_1")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request with invalid UUID
		reqURL := fmt.Sprintf("%s/ip-whitelist/invalid-uuid/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 400 because UUID is invalid
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_label_too_long_returns_400", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Set enterprise features: enabled=true
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

		// Use enterprise_1 who already has IP whitelist permissions
		email, password := findUserCredentials("enterprise_1")

		// Create request body with label longer than 100 characters
		longLabel := strings.Repeat("a", 256)
		requestBody := map[string]interface{}{
			"label": longLabel,
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request (using dummy ID)
		reqURL := fmt.Sprintf("%s/ip-whitelist/11111111-1111-1111-1111-111111111111/label/update", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 400 because label is too long
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_ip_rule_does_not_exist_returns_404", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Set enterprise features: enabled=true
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

		// Use enterprise_1 who already has IP whitelist permissions
		email, password := findUserCredentials("enterprise_1")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request with non-existent ID
		nonExistentID := "99999999-9999-9999-9999-999999999999"
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, nonExistentID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 404 because IP rule doesn't exist
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("test_different_tenant_ip_returns_404", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create IP rule for enterprise tenant
		ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Enterprise IP", setup.TestUsersData["enterprise_1"].UserID)

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

		// Use smb_1 who already has IP whitelist permissions
		email, password := findUserCredentials("smb_1")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Trying to update enterprise IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Try to update enterprise tenant's IP rule as SMB user
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, ipRuleID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login SMB user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 404 because IP belongs to different tenant
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("test_update_label_to_new_value_success", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

		// Set enterprise features: enabled=true
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

		// Create IP rule for this tenant
		ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Original Label", setup.TestUsersData["enterprise_1"].UserID)

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body
		requestBody := map[string]interface{}{
			"label": "Updated Label",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, ipRuleID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 200 because update succeeded
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_clear_label_empty_string_success", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

		// Set enterprise features: enabled=true
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

		// Create IP rule with a label
		ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Label to Clear", setup.TestUsersData["enterprise_1"].UserID)

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with empty label
		requestBody := map[string]interface{}{
			"label": "",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, ipRuleID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 200 because label was cleared
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_missing_label_field_success", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

		// Set enterprise features: enabled=true
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

		// Create IP rule with a label
		ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Label to Clear", setup.TestUsersData["enterprise_1"].UserID)

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body without label field
		requestBody := map[string]interface{}{
			// No label field
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, ipRuleID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 200 because missing field clears label
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_label_exactly_100_characters_success", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

		// Set enterprise features: enabled=true
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

		// Create IP rule
		ipRuleID := insertIPWhitelistRuleAndReturnID(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.0/24", "Original Label", setup.TestUsersData["enterprise_1"].UserID)

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with exactly 100 character label
		exactLabel := strings.Repeat("a", 100)
		requestBody := map[string]interface{}{
			"label": exactLabel,
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/%s/label/update", setup.BaseURL, ipRuleID)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Login user
		accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make request
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 200 because 100 characters is allowed
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
