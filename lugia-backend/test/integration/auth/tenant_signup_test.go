package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

type TenantSignupRequestBody struct {
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	CompanyName     string `json:"company_name"`
	UserName        string `json:"user_name"`
}

type CreateTenantTokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func generateValidJWTToken(t *testing.T, email string) string {
	t.Helper()
	
	// Use the same secret as the backend - this matches the docker-compose environment variable
	secret := []byte("test_create_tenant_jwt_secret_for_testing_only")
	
	now := time.Now()
	claims := CreateTenantTokenClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	assert.NoError(t, err, "Failed to generate JWT token")
	
	return tokenString
}

func generateExpiredJWTToken(t *testing.T, email string) string {
	t.Helper()
	
	secret := []byte("test_create_tenant_jwt_secret_for_testing_only")
	
	past := time.Now().Add(-1 * time.Hour)
	claims := CreateTenantTokenClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(past.Add(-48 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(past),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	assert.NoError(t, err, "Failed to generate expired JWT token")
	
	return tokenString
}

func generateJWTTokenWithWrongSecret(t *testing.T, email string) string {
	t.Helper()
	
	wrongSecret := []byte("wrong_secret")
	
	now := time.Now()
	claims := CreateTenantTokenClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(wrongSecret)
	assert.NoError(t, err, "Failed to generate JWT token with wrong secret")
	
	return tokenString
}

func generateJWTTokenWithWrongSigningMethod(t *testing.T, email string) string {
	t.Helper()
	
	// Return a manually crafted token with RS256 header but invalid signature
	// This will fail JWT validation because the server expects HS256
	return "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJleHAiOjk5OTk5OTk5OTl9.invalid_signature"
}

func makeTenantSignupRequest(t *testing.T, token string, requestBody TenantSignupRequestBody) *http.Response {
	t.Helper()
	
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err, "Failed to marshal request body")
	
	var requestURL string
	if token != "" {
		encodedToken := url.QueryEscape(token)
		requestURL = fmt.Sprintf("%s/auth/tenant-signup?token=%s", setup.BaseURL, encodedToken)
	} else {
		requestURL = fmt.Sprintf("%s/auth/tenant-signup", setup.BaseURL)
	}
	
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(body))
	assert.NoError(t, err, "Failed to create request")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Failed to execute request")
	
	return resp
}

func TestTenantSignupJWTValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	validRequestBody := TenantSignupRequestBody{
		Password:        "password123",
		PasswordConfirm: "password123",
		CompanyName:     "Test Company",
		UserName:        "Test User",
	}
	
	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid JWT token",
			token:          generateValidJWTToken(t, "new@example.com"),
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "missing token parameter",
			token:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
		{
			name:           "empty token parameter",
			token:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
		{
			name:           "malformed JWT token",
			token:          "invalid.jwt.token",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
		{
			name:           "JWT with invalid signature",
			token:          generateJWTTokenWithWrongSecret(t, "new@example.com"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
		{
			name:           "expired JWT token",
			token:          generateExpiredJWTToken(t, "new@example.com"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
		{
			name:           "JWT with unexpected signing method",
			token:          generateJWTTokenWithWrongSigningMethod(t, "new@example.com"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := makeTenantSignupRequest(t, tt.token, validRequestBody)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			
			if tt.expectedError != "" {
				var errorResponse map[string]any
				err := json.NewDecoder(resp.Body).Decode(&errorResponse)
				assert.NoError(t, err, "Failed to decode error response")
				assert.Equal(t, tt.expectedError, errorResponse["error"])
			}
			
			if tt.expectedStatus == http.StatusOK {
				// Check cookies are set
				cookies := resp.Cookies()
				assert.NotEmpty(t, cookies, "Expected cookies for successful signup")
				
				var accessToken, refreshToken *http.Cookie
				for _, cookie := range cookies {
					switch cookie.Name {
					case "dislyze_access_token":
						accessToken = cookie
					case "dislyze_refresh_token":
						refreshToken = cookie
					}
				}
				
				assert.NotNil(t, accessToken, "Access token cookie not found")
				assert.True(t, accessToken.HttpOnly, "Access token cookie should be HttpOnly")
				assert.True(t, accessToken.Secure, "Access token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, accessToken.SameSite, "Access token cookie should have SameSite=Strict")
				
				assert.NotNil(t, refreshToken, "Refresh token cookie not found")
				assert.True(t, refreshToken.HttpOnly, "Refresh token cookie should be HttpOnly")
				assert.True(t, refreshToken.Secure, "Refresh token cookie should be Secure")
				assert.Equal(t, http.SameSiteStrictMode, refreshToken.SameSite, "Refresh token cookie should have SameSite=Strict")
			} else {
				// Check no cookies are set for failed requests
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed signup")
			}
		})
	}
}

func TestTenantSignupJWTClaimsValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	validRequestBody := TenantSignupRequestBody{
		Password:        "password123",
		PasswordConfirm: "password123",
		CompanyName:     "Test Company",
		UserName:        "Test User",
	}
	
	tests := []struct {
		name           string
		email          string
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "JWT with empty email claim",
			email:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "無効または期限切れの招待リンクです。",
			description:    "Empty email in JWT should be rejected",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateValidJWTToken(t, tt.email)
			resp := makeTenantSignupRequest(t, token, validRequestBody)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)
			
			if tt.expectedError != "" {
				var errorResponse map[string]any
				err := json.NewDecoder(resp.Body).Decode(&errorResponse)
				assert.NoError(t, err, "Failed to decode error response")
				assert.Equal(t, tt.expectedError, errorResponse["error"])
			}
			
			if tt.expectedStatus != http.StatusOK {
				// Check no cookies are set for failed requests
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed signup")
			}
		})
	}
}

func TestTenantSignupRequestBodyValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	validToken := generateValidJWTToken(t, "new@example.com")
	
	tests := []struct {
		name           string
		requestBody    TenantSignupRequestBody
		expectedStatus int
	}{
		{
			name: "valid request body with all fields",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password123",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing password field",
			requestBody: TenantSignupRequestBody{
				PasswordConfirm: "password123",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password field",
			requestBody: TenantSignupRequestBody{
				Password:        "",
				PasswordConfirm: "password123",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password less than 8 characters",
			requestBody: TenantSignupRequestBody{
				Password:        "short",
				PasswordConfirm: "short",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password_confirm field",
			requestBody: TenantSignupRequestBody{
				Password:    "password123",
				CompanyName: "Test Company",
				UserName:    "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty password_confirm field",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password and password_confirm don't match",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password456",
				CompanyName:     "Test Company",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing company_name field",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password123",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty company_name field",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password123",
				CompanyName:     "",
				UserName:        "Test User",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing user_name field",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password123",
				CompanyName:     "Test Company",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty user_name field",
			requestBody: TenantSignupRequestBody{
				Password:        "password123",
				PasswordConfirm: "password123",
				CompanyName:     "Test Company",
				UserName:        "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := makeTenantSignupRequest(t, validToken, tt.requestBody)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			
			if tt.expectedStatus == http.StatusOK {
				// Check cookies are set
				cookies := resp.Cookies()
				assert.NotEmpty(t, cookies, "Expected cookies for successful signup")
			} else {
				// Check no cookies are set for failed requests
				cookies := resp.Cookies()
				assert.Empty(t, cookies, "Expected no cookies for failed signup")
			}
		})
	}
}

func TestTenantSignupInvalidJSON(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	validToken := generateValidJWTToken(t, "new@example.com")
	
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON request body",
			requestBody:    `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:           "empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encodedToken := url.QueryEscape(validToken)
			requestURL := fmt.Sprintf("%s/auth/tenant-signup?token=%s", setup.BaseURL, encodedToken)
			
			req, err := http.NewRequest("POST", requestURL, bytes.NewBufferString(tt.requestBody))
			assert.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")
			
			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request")
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			
			var errorResponse map[string]any
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			assert.NoError(t, err, "Failed to decode error response")
			assert.Equal(t, tt.expectedError, errorResponse["error"])
			
			// Check no cookies are set for failed requests
			cookies := resp.Cookies()
			assert.Empty(t, cookies, "Expected no cookies for failed signup")
		})
	}
}

func TestTenantSignupEmailCollision(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	// Use an email that already exists in the seeded data
	existingEmail := "enterprise1@localhost.com" // This exists in the seed data
	token := generateValidJWTToken(t, existingEmail)
	
	requestBody := TenantSignupRequestBody{
		Password:        "password123",
		PasswordConfirm: "password123",
		CompanyName:     "Test Company",
		UserName:        "Test User",
	}
	
	resp := makeTenantSignupRequest(t, token, requestBody)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()
	
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	
	var errorResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&errorResponse)
	assert.NoError(t, err, "Failed to decode error response")
	assert.Equal(t, "このメールアドレスは既に使用されています。", errorResponse["error"])
	
	// Check no cookies are set for failed request
	cookies := resp.Cookies()
	assert.Empty(t, cookies, "Expected no cookies for duplicate email signup")
}

func TestTenantSignupErrorResponseFormat(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)
	
	// Test with missing token to trigger error response
	requestBody := TenantSignupRequestBody{
		Password:        "password123",
		PasswordConfirm: "password123",
		CompanyName:     "Test Company",
		UserName:        "Test User",
	}
	
	resp := makeTenantSignupRequest(t, "", requestBody)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()
	
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	
	// Verify error response format matches project standard
	var errorResponse map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&errorResponse)
	assert.NoError(t, err, "Failed to decode error response")
	
	// Check that the response has the expected structure
	assert.Contains(t, errorResponse, "error", "Error response should contain 'error' field")
	assert.IsType(t, "", errorResponse["error"], "Error field should be a string")
	assert.NotEmpty(t, errorResponse["error"], "Error message should not be empty")
}