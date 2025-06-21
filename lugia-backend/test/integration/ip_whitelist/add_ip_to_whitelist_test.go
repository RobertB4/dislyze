package ip_whitelist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"lugia/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestAddIPToWhitelistIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	t.Run("test_unauthenticated_returns_401", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create request body
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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
			"ip_address": "192.168.1.100",
			"label":      "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

	t.Run("test_ip_whitelist_edit_permission_succeeds", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, "11111111-1111-1111-1111-111111111111", []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, "11111111-1111-1111-1111-111111111111")

		// Set enterprise features: enabled=true
		updateTenantEnterpriseFeatures(t, pool, "11111111-1111-1111-1111-111111111111", map[string]interface{}{
			"rbac": map[string]interface{}{
				"enabled": true,
			},
			"ip_whitelist": map[string]interface{}{
				"enabled":                     true,
				"active":                      false,
				"allow_internal_admin_bypass": false,
			},
		})

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because user has IP whitelist edit permission
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_feature_disabled_returns_403", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create malformed JSON body
		malformedJSON := `{"ip_address": "192.168.1.100", "label": "Test IP"`  // Missing closing brace

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

	t.Run("test_empty_request_body_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request with empty body
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer([]byte("")))
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

		// Check status code - should return 400 because request body is empty
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_missing_ip_address_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body without ip_address field
		requestBody := map[string]interface{}{
			"label": "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 400 because ip_address field is missing
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_empty_ip_address_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with empty ip_address
		requestBody := map[string]interface{}{
			"ip_address": "",
			"label":      "Test IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 400 because ip_address is empty
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_invalid_ip_address_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with invalid IP address
		requestBody := map[string]interface{}{
			"ip_address": "999.999.999.999",
			"label":      "Invalid IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 400 because IP address is invalid
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_invalid_cidr_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with invalid CIDR notation
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.0/99",
			"label":      "Invalid CIDR",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 400 because CIDR notation is invalid
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("test_add_ipv4_single_ip_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with IPv4 single IP (will be auto-converted to /32)
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Test IPv4 Single IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IP was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_add_ipv4_cidr_range_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with IPv4 CIDR range
		requestBody := map[string]interface{}{
			"ip_address": "172.16.0.0/12",
			"label":      "Test IPv4 CIDR Range",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because CIDR range was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_duplicate_ip_returns_400", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// First add an IP
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "First IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create first request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Make first request - should succeed
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Now try to add the same IP again (duplicate)
		duplicateRequestBody := map[string]interface{}{
			"ip_address": "192.168.1.100", // Same IP as above (will be normalized to 192.168.1.100/32)
			"label":      "Duplicate IP",
		}
		duplicateBodyBytes, err := json.Marshal(duplicateRequestBody)
		assert.NoError(t, err)

		// Create duplicate request
		duplicateReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(duplicateBodyBytes))
		assert.NoError(t, err)
		duplicateReq.Header.Set("Content-Type", "application/json")
		duplicateReq.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: accessToken,
			Path:  "/",
		})

		// Make duplicate request
		duplicateResp, err := client.Do(duplicateReq)
		assert.NoError(t, err)
		defer func() {
			if err := duplicateResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 400 because IP already exists
		assert.Equal(t, http.StatusBadRequest, duplicateResp.StatusCode)
	})

	t.Run("test_add_ipv6_single_ip_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with IPv6 single IP (will be auto-converted to /128)
		requestBody := map[string]interface{}{
			"ip_address": "2001:db8::1",
			"label":      "Test IPv6 Single IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IPv6 IP was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_add_ipv6_cidr_range_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with IPv6 CIDR range
		requestBody := map[string]interface{}{
			"ip_address": "2001:db8::/64",
			"label":      "Test IPv6 CIDR Range",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IPv6 CIDR range was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_add_with_label_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body with label
		requestBody := map[string]interface{}{
			"ip_address": "10.0.0.100",
			"label":      "Office Server",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IP with label was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_add_without_label_success", func(t *testing.T) {
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

		// Get user credentials
		email, password := findUserCredentials("enterprise_3")

		// Create request body without label (omit field entirely)
		requestBody := map[string]interface{}{
			"ip_address": "10.0.0.200",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IP without label was added successfully
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_tenant_isolation_add_to_correct_tenant", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise_3
		roleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, roleID, setup.TestTenantsData["enterprise"].ID)

		// Set enterprise features for enterprise tenant
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

		// Get user credentials (enterprise user)
		email, password := findUserCredentials("enterprise_3")

		// Create request body
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Enterprise IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		// Create request
		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
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

		// Check status code - should return 200 because IP was added to enterprise tenant
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("test_different_tenants_can_add_same_ip", func(t *testing.T) {
		// Reset database state before test
		setup.ResetAndSeedDB(t, pool)

		// Create role with ip_whitelist.edit permission for enterprise tenant
		enterpriseRoleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["enterprise"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["enterprise_3"].UserID, enterpriseRoleID, setup.TestTenantsData["enterprise"].ID)

		// Create role with ip_whitelist.edit permission for SMB tenant
		smbRoleID := createIPWhitelistRole(t, pool, setup.TestTenantsData["smb"].ID, []string{
			"a9b8c7d6-e5f4-a3b2-c1d0-e9f8a7b6c5d4", // ip_whitelist.edit permission
		})
		assignRoleToUser(t, pool, setup.TestUsersData["smb_1"].UserID, smbRoleID, setup.TestTenantsData["smb"].ID)

		// Set enterprise features for both tenants
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

		// First, add IP to enterprise tenant
		enterpriseEmail, enterprisePassword := findUserCredentials("enterprise_3")
		requestBody := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"label":      "Enterprise IP",
		}
		bodyBytes, err := json.Marshal(requestBody)
		assert.NoError(t, err)

		reqURL := fmt.Sprintf("%s/ip-whitelist/create", setup.BaseURL)
		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(bodyBytes))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		enterpriseAccessToken, _ := setup.LoginUserAndGetTokens(t, enterpriseEmail, enterprisePassword)
		req.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: enterpriseAccessToken,
			Path:  "/",
		})

		// Make first request - should succeed
		resp, err := client.Do(req)
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Now add the same IP to SMB tenant - should also succeed
		smbEmail, smbPassword := findUserCredentials("smb_1")
		smbRequestBody := map[string]interface{}{
			"ip_address": "192.168.1.100", // Same IP as enterprise tenant
			"label":      "SMB IP",
		}
		smbBodyBytes, err := json.Marshal(smbRequestBody)
		assert.NoError(t, err)

		smbReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(smbBodyBytes))
		assert.NoError(t, err)
		smbReq.Header.Set("Content-Type", "application/json")

		smbAccessToken, _ := setup.LoginUserAndGetTokens(t, smbEmail, smbPassword)
		smbReq.AddCookie(&http.Cookie{
			Name:  "dislyze_access_token",
			Value: smbAccessToken,
			Path:  "/",
		})

		// Make SMB request
		smbResp, err := client.Do(smbReq)
		assert.NoError(t, err)
		defer func() {
			if err := smbResp.Body.Close(); err != nil {
				t.Logf("Error closing response body: %v", err)
			}
		}()

		// Check status code - should return 200 because same IP can exist in different tenants
		assert.Equal(t, http.StatusOK, smbResp.StatusCode)
	})
}
