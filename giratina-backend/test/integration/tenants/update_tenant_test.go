package tenants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"giratina/features/tenants"
	"giratina/test/integration/setup"
	"io"
	"net/http"
	"testing"

	"dislyze/jirachi/authz"
	"github.com/stretchr/testify/assert"
)

func TestUpdateTenant_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	validTenantID := "33333333-3333-3333-3333-333333333333" // internal tenant from seed data

	validUpdateRequest := tenants.UpdateTenantRequestBody{
		Name: "更新されたテナント名",
		EnterpriseFeatures: authz.EnterpriseFeatures{
			RBAC: authz.RBAC{Enabled: true},
		},
	}

	tests := []struct {
		name                string
		loginUserKey        string // Key for setup.TestUsersData map, empty for no login
		tenantID            string
		requestBody         any // Can be UpdateTenantRequestBody or raw string/nil
		expectedStatus      int
		expectErrorResponse bool
		validateResponse    func(t *testing.T, body []byte)
	}{
		// Security - Authentication & Authorization
		{
			name:                "unauthenticated request returns 401",
			loginUserKey:        "", // No login
			tenantID:            validTenantID,
			requestBody:         validUpdateRequest,
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
		},
		// NOTE: Removed "enterprise user (non-admin)" test because enterprise users cannot login to giratina (internal admin only app)
		{
			name:           "internal admin 1 succeeds",
			loginUserKey:   "internal_1",
			tenantID:       validTenantID,
			requestBody:    validUpdateRequest,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},
		{
			name:           "internal admin 2 succeeds",
			loginUserKey:   "internal_2",
			tenantID:       validTenantID,
			requestBody:    validUpdateRequest,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},

		// Security - Data Access Control
		{
			name:         "updating non-existent tenant succeeds (SQL UPDATE affects 0 rows but no error)",
			loginUserKey: "internal_1",
			tenantID:     "99999999-9999-9999-9999-999999999999",
			requestBody:  validUpdateRequest,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},

		// Edge Cases - Request Format
		{
			name:                "invalid JSON body returns 400",
			loginUserKey:        "internal_1",
			tenantID:            validTenantID,
			requestBody:         `{"name": "テストテナント", "enterprise_features":}`, // Invalid JSON
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:                "empty request body returns validation error",
			loginUserKey:        "internal_1",
			tenantID:            validTenantID,
			requestBody:         nil,
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},

		// Edge Cases - Field Validation
		{
			name:         "missing name field returns 400",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"enterprise_features": map[string]any{
					"rbac": map[string]any{"enabled": true},
				},
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "empty name field returns 400",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: tenants.UpdateTenantRequestBody{
				Name: "",
				EnterpriseFeatures: authz.EnterpriseFeatures{
					RBAC: authz.RBAC{Enabled: true},
				},
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "whitespace-only name returns 400",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: tenants.UpdateTenantRequestBody{
				Name: "   ",
				EnterpriseFeatures: authz.EnterpriseFeatures{
					RBAC: authz.RBAC{Enabled: true},
				},
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "missing enterprise_features field succeeds (gets zero value)",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name": "テストテナント",
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},

		// Edge Cases - Enterprise Features Validation
		{
			name:         "invalid enterprise_features JSON returns 400",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name":                "テストテナント",
				"enterprise_features": "invalid_json_structure",
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "unknown enterprise feature succeeds (ignored by JSON unmarshaling)",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name": "テストテナント",
				"enterprise_features": map[string]any{
					"unknown_feature": map[string]any{"enabled": true},
					"rbac":            map[string]any{"enabled": true},
				},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},
		{
			name:         "invalid rbac structure succeeds (missing fields ignored)",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name": "テストテナント",
				"enterprise_features": map[string]any{
					"rbac": map[string]any{"invalid_field": true},
				},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},
		{
			name:         "invalid rbac.enabled type (non-boolean) returns 400",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name": "テストテナント",
				"enterprise_features": map[string]any{
					"rbac": map[string]any{"enabled": "true"}, // String instead of bool
				},
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "extra fields in rbac succeed (ignored by JSON unmarshaling)",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: map[string]any{
				"name": "テストテナント",
				"enterprise_features": map[string]any{
					"rbac": map[string]any{
						"enabled":     true,
						"extra_field": "value",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},

		// Edge Cases - URL Parameters
		{
			name:                "invalid tenant ID format returns 400",
			loginUserKey:        "internal_1",
			tenantID:            "invalid-uuid",
			requestBody:         validUpdateRequest,
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:                "partial UUID returns 400",
			loginUserKey:        "internal_1",
			tenantID:            "33333333-3333-4444", // Partial UUID
			requestBody:         validUpdateRequest,
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},

		// Positive Cases - Successful Updates
		{
			name:         "valid update with rbac enabled",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: tenants.UpdateTenantRequestBody{
				Name: "RBACテナント",
				EnterpriseFeatures: authz.EnterpriseFeatures{
					RBAC: authz.RBAC{Enabled: true},
				},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},
		{
			name:         "valid update with rbac disabled",
			loginUserKey: "internal_1",
			tenantID:     validTenantID,
			requestBody: tenants.UpdateTenantRequestBody{
				Name: "非RBACテナント",
				EnterpriseFeatures: authz.EnterpriseFeatures{
					RBAC: authz.RBAC{Enabled: false},
				},
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.UpdateTenantResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "Tenant updated successfully", response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset database for each test to ensure clean state
			setup.ResetAndSeedDB(t, pool)

			var cookies []*http.Cookie

			if tt.loginUserKey != "" {
				currentUserDetails, ok := setup.TestUsersData[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: User key '%s' not found in TestUsersData", tt.loginUserKey)
				}

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			// Prepare request body
			var requestBody []byte
			var err error

			if tt.requestBody == nil {
				requestBody = []byte("{}")
			} else if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			url := fmt.Sprintf("%s/tenants/%s/update", setup.BaseURL, tt.tenantID)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			if len(cookies) > 0 {
				for _, cookie := range cookies {
					req.AddCookie(cookie)
				}
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)

			if tt.validateResponse != nil {
				tt.validateResponse(t, bodyBytes)
			} else if tt.expectErrorResponse {
				t.Logf("Received error response body for %s: %s", tt.name, string(bodyBytes))
			}
		})
	}
}

func TestUpdateTenant_DatabaseChanges(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	validTenantID := "33333333-3333-3333-3333-333333333333"

	// Login as admin
	currentUserDetails := setup.TestUsersData["internal_1"]
	accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
	cookies := []*http.Cookie{
		{Name: "dislyze_access_token", Value: accessToken},
		{Name: "dislyze_refresh_token", Value: refreshToken},
	}

	updateRequest := tenants.UpdateTenantRequestBody{
		Name: "データベース更新テスト",
		EnterpriseFeatures: authz.EnterpriseFeatures{
			RBAC: authz.RBAC{Enabled: false}, // Change from default true
		},
	}

	requestBody, err := json.Marshal(updateRequest)
	assert.NoError(t, err)

	// Perform update
	url := fmt.Sprintf("%s/tenants/%s/update", setup.BaseURL, validTenantID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify database changes by getting the tenant
	getReq, err := http.NewRequest("GET", fmt.Sprintf("%s/tenants", setup.BaseURL), nil)
	assert.NoError(t, err)

	for _, cookie := range cookies {
		getReq.AddCookie(cookie)
	}

	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	defer func() {
		if err := getResp.Body.Close(); err != nil {
			t.Logf("Error closing get response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var tenantsResponse tenants.GetTenantsResponse
	err = json.NewDecoder(getResp.Body).Decode(&tenantsResponse)
	assert.NoError(t, err)

	// Find the updated tenant
	var updatedTenant *tenants.TenantResponse
	for _, tenant := range tenantsResponse.Tenants {
		if tenant.ID == validTenantID {
			updatedTenant = &tenant
			break
		}
	}

	assert.NotNil(t, updatedTenant, "Updated tenant not found in response")
	assert.Equal(t, "データベース更新テスト", updatedTenant.Name)
	assert.False(t, updatedTenant.EnterpriseFeatures.RBAC.Enabled)
}

func TestUpdateTenant_ExistingTenantFromSeedData(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	// Use the known tenant from seed data
	seedTenant := setup.TestTenantsData["internal"]
	
	// Login as admin
	currentUserDetails := setup.TestUsersData["internal_1"]
	accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
	cookies := []*http.Cookie{
		{Name: "dislyze_access_token", Value: accessToken},
		{Name: "dislyze_refresh_token", Value: refreshToken},
	}

	updateRequest := tenants.UpdateTenantRequestBody{
		Name: "シードデータテナント更新",
		EnterpriseFeatures: authz.EnterpriseFeatures{
			RBAC: authz.RBAC{Enabled: true},
		},
	}

	requestBody, err := json.Marshal(updateRequest)
	assert.NoError(t, err)

	url := fmt.Sprintf("%s/tenants/%s/update", setup.BaseURL, seedTenant.ID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var response tenants.UpdateTenantResponse
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err)
	assert.Equal(t, "Tenant updated successfully", response.Message)
}