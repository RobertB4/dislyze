package ip_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	jwtlib "lugia/lib/jwt"
	"lugia/test/integration/setup"

	"dislyze/jirachi/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEmergencyJWTSecret = "test_ip_whitelist_emergency_jwt_secret"

// Helper function to generate valid emergency token and insert JTI in database
func generateValidEmergencyTokenWithDB(t *testing.T, pool *pgxpool.Pool, userID, tenantID string) string {
	userUUID := pgtype.UUID{}
	err := userUUID.Scan(userID)
	require.NoError(t, err)

	tenantUUID := pgtype.UUID{}
	err = tenantUUID.Scan(tenantID)
	require.NoError(t, err)

	tokenString, jti, err := jwtlib.GenerateEmergencyToken(userUUID, tenantUUID, []byte(testEmergencyJWTSecret))
	require.NoError(t, err)

	// Insert JTI into database
	ctx := context.Background()
	_, err = pool.Exec(ctx, "INSERT INTO ip_whitelist_emergency_tokens (jti) VALUES ($1)", jti)
	require.NoError(t, err)

	return tokenString
}

// Helper function to generate expired emergency token
func generateExpiredEmergencyToken(t *testing.T, userID, tenantID string) string {
	userUUID := pgtype.UUID{}
	err := userUUID.Scan(userID)
	require.NoError(t, err)

	tenantUUID := pgtype.UUID{}
	err = tenantUUID.Scan(tenantID)
	require.NoError(t, err)

	jti, err := utils.NewUUID()
	require.NoError(t, err)

	// Create token that expired 1 hour ago
	pastTime := time.Now().Add(-1 * time.Hour)
	claims := jwtlib.EmergencyClaims{
		UserID:   userUUID,
		TenantID: tenantUUID,
		Action:   "ip_whitelist.emergency_deactivate",
		JTI:      jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(pastTime.Add(-30 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(pastTime), // Expired
			NotBefore: jwt.NewNumericDate(pastTime.Add(-30 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testEmergencyJWTSecret))
	require.NoError(t, err)

	return tokenString
}

// Helper function to generate token with wrong action
func generateWrongActionToken(t *testing.T, userID, tenantID string) string {
	userUUID := pgtype.UUID{}
	err := userUUID.Scan(userID)
	require.NoError(t, err)

	tenantUUID := pgtype.UUID{}
	err = tenantUUID.Scan(tenantID)
	require.NoError(t, err)

	jti, err := utils.NewUUID()
	require.NoError(t, err)

	now := time.Now()
	claims := jwtlib.EmergencyClaims{
		UserID:   userUUID,
		TenantID: tenantUUID,
		Action:   "wrong_action", // Wrong action
		JTI:      jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testEmergencyJWTSecret))
	require.NoError(t, err)

	return tokenString
}

// Helper function to generate token with wrong signing key
func generateWrongKeyToken(t *testing.T, userID, tenantID string) string {
	userUUID := pgtype.UUID{}
	err := userUUID.Scan(userID)
	require.NoError(t, err)

	tenantUUID := pgtype.UUID{}
	err = tenantUUID.Scan(tenantID)
	require.NoError(t, err)

	jti, err := utils.NewUUID()
	require.NoError(t, err)

	now := time.Now()
	claims := jwtlib.EmergencyClaims{
		UserID:   userUUID,
		TenantID: tenantUUID,
		Action:   "ip_whitelist.emergency_deactivate",
		JTI:      jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong_secret_key"))
	require.NoError(t, err)

	return tokenString
}

// Helper function to mark emergency token as used
func markEmergencyTokenAsUsed(t *testing.T, pool *pgxpool.Pool, jtiString string) {
	jti := pgtype.UUID{}
	err := jti.Scan(jtiString)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = pool.Exec(ctx, "UPDATE ip_whitelist_emergency_tokens SET used_at = CURRENT_TIMESTAMP WHERE jti = $1", jti)
	require.NoError(t, err)
}

func TestEmergencyDeactivateIntegration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)
	client := &http.Client{}

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, pool *pgxpool.Pool) (userKey, token string, skipIPAdd bool)
		expectedStatus int
		validateFunc   func(t *testing.T, pool *pgxpool.Pool)
	}{
		{
			name: "test_unauthenticated_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
				return "", "", true // No user login
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_no_ip_whitelist_permission_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				token := generateValidEmergencyTokenWithDB(t, pool, setup.TestUsersData["enterprise_2"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_2", token, true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_missing_token_parameter_returns_400",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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
				return "enterprise_1", "", true // No token
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "test_empty_token_parameter_returns_400",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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
				return "enterprise_1", "", true // Empty token (same as missing)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "test_invalid_jwt_token_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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
				return "enterprise_1", "invalid-jwt-string", true
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_expired_jwt_token_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				token := generateExpiredEmergencyToken(t, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_wrong_action_token_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				token := generateWrongActionToken(t, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_wrong_signing_key_returns_401",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				token := generateWrongKeyToken(t, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "test_token_user_mismatch_returns_403",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				// Generate token for enterprise_2 but authenticate as enterprise_1
				token := generateValidEmergencyTokenWithDB(t, pool, setup.TestUsersData["enterprise_2"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "test_token_already_used_returns_409",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
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

				token := generateValidEmergencyTokenWithDB(t, pool, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)

				// Mark token as used
				// Extract JTI from token to mark as used
				parsedToken, err := jwt.ParseWithClaims(token, &jwtlib.EmergencyClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(testEmergencyJWTSecret), nil
				})
				require.NoError(t, err)
				claims := parsedToken.Claims.(*jwtlib.EmergencyClaims)
				markEmergencyTokenAsUsed(t, pool, claims.JTI.String())

				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "test_valid_token_deactivates_successfully",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["enterprise"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": true,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true, // Currently active
						"allow_internal_admin_bypass": false,
					},
				})

				token := generateValidEmergencyTokenWithDB(t, pool, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "test_tenant_isolation",
			setupFunc: func(t *testing.T, pool *pgxpool.Pool) (string, string, bool) {
				// Set up both enterprise and SMB tenants with active whitelists
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
				updateTenantEnterpriseFeatures(t, pool, setup.TestTenantsData["smb"].ID, map[string]interface{}{
					"rbac": map[string]interface{}{
						"enabled": false,
					},
					"ip_whitelist": map[string]interface{}{
						"enabled":                     true,
						"active":                      true, // SMB also has active whitelist
						"allow_internal_admin_bypass": false,
					},
				})

				token := generateValidEmergencyTokenWithDB(t, pool, setup.TestUsersData["enterprise_1"].UserID, setup.TestTenantsData["enterprise"].ID)
				return "enterprise_1", token, true
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, pool *pgxpool.Pool) {
				// Verify enterprise tenant is deactivated
				var enterpriseFeatures map[string]interface{}
				row := pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["enterprise"].ID)
				var featuresJSON []byte
				err := row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &enterpriseFeatures)
				require.NoError(t, err)
				assert.False(t, enterpriseFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))

				// Verify SMB tenant remains active
				var smbFeatures map[string]interface{}
				row = pool.QueryRow(context.Background(),
					"SELECT enterprise_features FROM tenants WHERE id = $1",
					setup.TestTenantsData["smb"].ID)
				err = row.Scan(&featuresJSON)
				require.NoError(t, err)
				err = json.Unmarshal(featuresJSON, &smbFeatures)
				require.NoError(t, err)
				assert.True(t, smbFeatures["ip_whitelist"].(map[string]interface{})["active"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup.ResetAndSeedDB(t, pool)

			userKey, token, skipIPAdd := tt.setupFunc(t, pool)

			// Add user IP to whitelist if needed (so middleware allows request)
			// Note: Emergency endpoint bypasses IP whitelist, but we still add for consistency
			if !skipIPAdd && userKey != "" {
				insertIPWhitelistRule(t, pool, setup.TestTenantsData["enterprise"].ID, "192.168.1.100", "Test IP", setup.TestUsersData[userKey].UserID)
			}

			// Create request URL with token parameter
			var reqURL string
			if token != "" {
				reqURL = fmt.Sprintf("%s/ip-whitelist/emergency-deactivate?token=%s", setup.BaseURL, token)
			} else {
				reqURL = fmt.Sprintf("%s/ip-whitelist/emergency-deactivate", setup.BaseURL)
			}

			req, err := http.NewRequest("POST", reqURL, nil)
			require.NoError(t, err)

			// Set client IP header (doesn't matter much since emergency bypasses, but for consistency)
			req.Header.Set("X-Real-IP", "192.168.1.100")

			// Add auth if user specified
			if userKey != "" {
				email, password := findUserCredentials(userKey)
				accessToken, _ := setup.LoginUserAndGetTokens(t, email, password)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			// Verify status
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Run additional validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, pool)
			}
		})
	}
}
