package tenants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"giratina/features/tenants"
	"giratina/test/integration/setup"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTenantInvitationToken_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	validEmail := "test@example.com"

	validRequest := tenants.GenerateTenantInvitationTokenRequest{
		Email: validEmail,
	}

	tests := []struct {
		name                string
		loginUserKey        string // Key for setup.TestUsersData map, empty for no login
		requestBody         any    // Can be GenerateTenantInvitationTokenRequest or raw string/nil
		expectedStatus      int
		expectErrorResponse bool
		validateResponse    func(t *testing.T, body []byte)
	}{
		// Security - Authentication
		{
			name:                "unauthenticated request returns 401",
			loginUserKey:        "", // No login
			requestBody:         validRequest,
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
		},
		{
			name:           "internal admin 1 succeeds",
			loginUserKey:   "internal_1",
			requestBody:    validRequest,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.GenerateTenantInvitationTokenResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.URL, "URL should not be empty")
			},
		},
		{
			name:           "internal admin 2 succeeds",
			loginUserKey:   "internal_2",
			requestBody:    validRequest,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.GenerateTenantInvitationTokenResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.URL, "URL should not be empty")
			},
		},

		// Request Body Validation
		{
			name:         "valid email succeeds",
			loginUserKey: "internal_1",
			requestBody: tenants.GenerateTenantInvitationTokenRequest{
				Email: "valid@example.com",
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body []byte) {
				var response tenants.GenerateTenantInvitationTokenResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.URL, "URL should not be empty")
			},
		},
		{
			name:         "empty email returns 400",
			loginUserKey: "internal_1",
			requestBody: tenants.GenerateTenantInvitationTokenRequest{
				Email: "",
			},
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:         "missing email field returns 400",
			loginUserKey: "internal_1",
			requestBody:  map[string]any{}, // Empty object
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:                "invalid JSON returns 400",
			loginUserKey:        "internal_1",
			requestBody:         `{"email": "test@example.com"`, // Missing closing brace
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
		},
		{
			name:                "empty request body returns 400",
			loginUserKey:        "internal_1",
			requestBody:         nil,
			expectedStatus:      http.StatusBadRequest,
			expectErrorResponse: true,
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

			url := fmt.Sprintf("%s/tenants/generate-token", setup.BaseURL)
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

func TestGenerateTenantInvitationToken_JWTValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	testEmail := "jwt-test@example.com"

	// Login as admin
	currentUserDetails := setup.TestUsersData["internal_1"]
	accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
	cookies := []*http.Cookie{
		{Name: "dislyze_access_token", Value: accessToken},
		{Name: "dislyze_refresh_token", Value: refreshToken},
	}

	request := tenants.GenerateTenantInvitationTokenRequest{
		Email: testEmail,
	}

	requestBody, err := json.Marshal(request)
	assert.NoError(t, err)

	// Make request
	url := fmt.Sprintf("%s/tenants/generate-token", setup.BaseURL)
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

	var response tenants.GenerateTenantInvitationTokenResponse
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.URL, "URL should not be empty")

	// Extract JWT token from URL
	// URL format: {FrontendURL}/signup?token={jwt}
	parts := strings.Split(response.URL, "?token=")
	assert.Len(t, parts, 2, "URL should contain token parameter")
	jwtToken := parts[1]
	assert.NotEmpty(t, jwtToken, "JWT token should not be empty")

	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(jwtToken, &tenants.TenantInvitationClaims{}, func(token *jwt.Token) (any, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key from test environment
		return []byte("test_create_tenant_jwt_secret_for_testing_only"), nil
	})

	assert.NoError(t, err, "JWT should be valid and parseable")
	assert.True(t, token.Valid, "JWT should be valid")

	// Validate claims
	claims, ok := token.Claims.(*tenants.TenantInvitationClaims)
	assert.True(t, ok, "Should be able to cast to TenantInvitationClaims")
	assert.Equal(t, testEmail, claims.Email, "Email in JWT should match request")

	// Validate expiration (should be approximately 48 hours from now)
	now := time.Now()
	expectedExpiry := now.Add(48 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time

	// Allow 1 minute tolerance for test execution time
	timeDiff := actualExpiry.Sub(expectedExpiry)
	assert.True(t, timeDiff >= -time.Minute && timeDiff <= time.Minute,
		"JWT expiration should be approximately 48 hours from now. Expected: %v, Actual: %v, Diff: %v",
		expectedExpiry, actualExpiry, timeDiff)

	t.Logf("JWT validation successful - Email: %s, Expires: %v", claims.Email, claims.ExpiresAt.Time)
}