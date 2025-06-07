package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecuritySQLInjectionProtection_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as alpha_admin
	accessToken, _ := setup.LoginUserAndGetTokens(t, setup.TestUsersData["alpha_admin"].Email, setup.TestUsersData["alpha_admin"].PlainTextPassword)

	tests := []struct {
		name     string
		endpoint string
		payload  interface{}
	}{
		{
			name:     "SQL injection in change name",
			endpoint: "/me/change-name",
			payload: map[string]string{
				"name": "'; DROP TABLE users; --",
			},
		},
		{
			name:     "SQL injection in change email",
			endpoint: "/me/change-email",
			payload: map[string]string{
				"new_email": "evil'; DROP TABLE users; --@example.com",
			},
		},
		{
			name:     "SQL injection in tenant name change",
			endpoint: "/tenant/change-name",
			payload: map[string]string{
				"name": "'; UPDATE users SET role='admin' WHERE email='alpha_editor@example.com'; --",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make the request with SQL injection payload
			reqBody, _ := json.Marshal(tt.payload)
			req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", setup.BaseURL, tt.endpoint), bytes.NewBuffer(reqBody))
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Verify the application doesn't crash and returns proper error
			// Should get 400 (validation error) or 200 (successful but safe processing)
			assert.True(t, resp.StatusCode == 400 || resp.StatusCode == 200,
				"Expected 400 or 200, got %d. SQL injection may have caused unexpected behavior", resp.StatusCode)

			// Verify database integrity - check that users table still exists and data is intact
			var userCount int
			err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&userCount)
			assert.NoError(t, err, "Users table should still exist and be accessible")
			assert.Greater(t, userCount, 0, "Users table should still contain data")

			// Verify no unauthorized privilege escalation occurred
			var alphaEditorRole string
			err = pool.QueryRow(context.Background(),
				"SELECT role FROM users WHERE email = $1",
				setup.TestUsersData["alpha_editor"].Email).Scan(&alphaEditorRole)
			assert.NoError(t, err)
			assert.Equal(t, "editor", alphaEditorRole, "Alpha editor role should not have been modified by SQL injection")
		})
	}
}

func TestSecurityXSSProtection_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as alpha_admin
	accessToken, _ := setup.LoginUserAndGetTokens(t, setup.TestUsersData["alpha_admin"].Email, setup.TestUsersData["alpha_admin"].PlainTextPassword)

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<svg onload=alert('xss')>",
		"'; alert('xss'); //",
		"<iframe src=javascript:alert('xss')></iframe>",
	}

	for _, payload := range xssPayloads {
		t.Run(fmt.Sprintf("XSS protection for payload: %s", payload[:min(20, len(payload))]), func(t *testing.T) {
			// Test XSS in user name change
			reqBody, _ := json.Marshal(map[string]string{
				"name": payload,
			})
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-name", setup.BaseURL), bytes.NewBuffer(reqBody))
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Should either reject the payload or safely store it
			if resp.StatusCode == 200 {
				// If accepted, verify the payload was safely stored (not executed)
				var storedName string
				err = pool.QueryRow(context.Background(),
					"SELECT name FROM users WHERE email = $1",
					setup.TestUsersData["alpha_admin"].Email).Scan(&storedName)
				assert.NoError(t, err)

				// The stored name should be the exact payload (safely stored as text)
				// but should not contain executable script elements when rendered
				assert.Equal(t, payload, storedName, "XSS payload should be stored as-is (safe text)")
			}

			// Test XSS in tenant name change
			reqBody, _ = json.Marshal(map[string]string{
				"name": payload,
			})
			req, err = http.NewRequest("POST", fmt.Sprintf("%s/tenant/change-name", setup.BaseURL), bytes.NewBuffer(reqBody))
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err = client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Should either reject the payload or safely store it
			assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 400,
				"XSS payload should be safely handled, got status %d", resp.StatusCode)
		})
	}
}

func TestSecurityHorizontalPrivilegeEscalation_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as alpha_editor (normal user)
	editorAccessToken, _ := setup.LoginUserAndGetTokens(t, setup.TestUsersData["alpha_editor"].Email, setup.TestUsersData["alpha_editor"].PlainTextPassword)

	tests := []struct {
		name           string
		endpoint       string
		method         string
		payload        interface{}
		expectedStatus int
		description    string
	}{
		{
			name:           "Cannot access admin-only users list",
			endpoint:       "/users",
			method:         "GET",
			payload:        nil,
			expectedStatus: 403, // Should be forbidden (admin-only endpoint)
			description:    "User should not be able to access admin-only users endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.payload != nil {
				reqBody, _ := json.Marshal(tt.payload)
				req, err = http.NewRequest(tt.method, fmt.Sprintf("%s%s", setup.BaseURL, tt.endpoint), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, fmt.Sprintf("%s%s", setup.BaseURL, tt.endpoint), nil)
			}
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: editorAccessToken,
				Path:  "/",
			})

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)

			// Additional verification: ensure admin user's data was not modified
			if strings.Contains(tt.name, "change") && resp.StatusCode == 200 {
				// Verify admin user's name was not changed
				var adminName string
				err = pool.QueryRow(context.Background(),
					"SELECT name FROM users WHERE id = $1",
					setup.TestUsersData["alpha_admin"].UserID).Scan(&adminName)
				assert.NoError(t, err)
				assert.Equal(t, setup.TestUsersData["alpha_admin"].Name, adminName,
					"Admin user's name should not have been modified by another user")
			}
		})
	}
}

func TestSecurityJWTSecurity_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.ResetAndSeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Login as alpha_editor to get valid access token
	editorAccessToken, _ := setup.LoginUserAndGetTokens(t, setup.TestUsersData["alpha_editor"].Email, setup.TestUsersData["alpha_editor"].PlainTextPassword)

	tests := []struct {
		name             string
		tokenManipulator func(string) string
		expectedStatus   int
		description      string
	}{
		{
			name: "Tampered JWT access token should be rejected",
			tokenManipulator: func(token string) string {
				// Tamper with the JWT by modifying the payload
				parts := strings.Split(token, ".")
				if len(parts) == 3 {
					// Modify the payload (base64 decode, change, re-encode)
					payload := parts[1]
					// Add padding if needed
					for len(payload)%4 != 0 {
						payload += "="
					}
					decoded, _ := base64.URLEncoding.DecodeString(payload)
					tampered := strings.Replace(string(decoded), "editor", "admin", 1)
					parts[1] = base64.URLEncoding.EncodeToString([]byte(tampered))
					return strings.Join(parts, ".")
				}
				return token
			},
			expectedStatus: 401,
			description:    "Tampered JWT should be rejected",
		},
		{
			name: "Missing JWT signature should be rejected",
			tokenManipulator: func(token string) string {
				// Remove signature part
				parts := strings.Split(token, ".")
				if len(parts) == 3 {
					return parts[0] + "." + parts[1] + "."
				}
				return token
			},
			expectedStatus: 401,
			description:    "JWT without signature should be rejected",
		},
		{
			name: "Invalid JWT format should be rejected",
			tokenManipulator: func(token string) string {
				return "invalid.jwt.token.format.extra"
			},
			expectedStatus: 401,
			description:    "Invalid JWT format should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply token manipulation
			tamperedToken := tt.tokenManipulator(editorAccessToken)

			// Try to access a protected endpoint
			req, err := http.NewRequest("POST", fmt.Sprintf("%s/me/change-name", setup.BaseURL),
				bytes.NewBuffer([]byte(`{"name":"Should Not Work"}`)))
			assert.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: tamperedToken,
				Path:  "/",
			})

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
