package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"lugia/handlers"
	"lugia/queries_pregeneration"
	"lugia/test/integration/setup"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestGetUsers_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name               string
		loginUserKey       string // Key for setup.TestUsersData map
		expectedStatus     int
		expectedUserEmails []string
		expectUnauth       bool
	}{
		{
			name:           "unauthenticated user gets 401",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "alpha_admin (Tenant A) gets users from Tenant Alpha",
			loginUserKey:   "alpha_admin",
			expectedStatus: http.StatusOK,
			// Order by created_at DESC from seed.sql
			expectedUserEmails: []string{
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:           "alpha_editor (Tenant A) gets forbidden because they are not an admin",
			loginUserKey:   "alpha_editor",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:               "beta_admin (Tenant B) gets users from Tenant Beta (only self)",
			loginUserKey:       "beta_admin",
			expectedStatus:     http.StatusOK,
			expectedUserEmails: []string{setup.TestUsersData["beta_admin"].Email},
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=1&limit=50", setup.BaseURL)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			if !tt.expectUnauth {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse handlers.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, 1, usersResponse.Pagination.Page, "Pagination page mismatch for test: %s", tt.name)
				assert.Equal(t, 50, usersResponse.Pagination.Limit, "Pagination limit mismatch for test: %s", tt.name)
				assert.Equal(t, len(tt.expectedUserEmails), usersResponse.Pagination.Total, "Pagination total mismatch for test: %s", tt.name)

				assert.Equal(t, len(tt.expectedUserEmails), len(usersResponse.Users), "Number of users mismatch for test: %s", tt.name)

				actualEmails := make([]string, len(usersResponse.Users))
				for i, u := range usersResponse.Users {
					actualEmails[i] = u.Email
					assert.NotEmpty(t, u.ID, "User ID should not be empty for user %s", u.Email)

					var expectedName, expectedUserID, expectedStatus string
					var expectedRole queries_pregeneration.UserRole
					foundInTestData := false
					for _, seededUser := range setup.TestUsersData {
						if seededUser.Email == u.Email {
							expectedName = seededUser.Name
							expectedRole = seededUser.Role
							expectedUserID = seededUser.UserID
							expectedStatus = seededUser.Status
							foundInTestData = true
							break
						}
					}
					assert.True(t, foundInTestData, "User with email %s not found in setup.TestUsersData. Check setup.sql and setup.TestUsersData map.", u.Email)
					assert.Equal(t, expectedUserID, u.ID, "ID mismatch for user %s", u.Email)
					assert.Equal(t, expectedName, u.Name, "Name mismatch for user %s", u.Email)
					assert.Equal(t, expectedRole, u.Role, "Role mismatch for user %s", u.Email)
					assert.Equal(t, expectedStatus, u.Status, "Status mismatch for user %s", u.Email)
				}
				assert.Equal(t, tt.expectedUserEmails, actualEmails, "User email list or order mismatch for test: %s", tt.name)
			}
		})
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type testUserDetail struct {
	Email             string
	PlainTextPassword string
	UserID            string
	TenantID          string
	Name              string
	Role              queries_pregeneration.UserRole
	Status            string
}

var expectedInviteErrorMessages = map[string]string{
	"emailConflict": "このメールアドレスは既に使用されています。",
}

func TestInviteUser_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)

	type inviteUserTestCase struct {
		name             string
		loginUserKey     string // Key for setup.TestUsersData map, empty for unauth
		requestBody      handlers.InviteUserRequest
		expectedStatus   int
		expectedErrorKey string // Key for expectedInviteErrorMessages, if any
		expectUnauth     bool
	}

	tests := []inviteUserTestCase{
		{
			name:         "successful invitation by alpha_admin",
			loginUserKey: "alpha_admin",
			requestBody: handlers.InviteUserRequest{
				Email: "new_invitee@example.com",
				Name:  "New Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:         "error when email already exists (alpha_admin invites existing alpha_editor)",
			loginUserKey: "alpha_admin",
			requestBody: handlers.InviteUserRequest{
				Email: setup.TestUsersData["alpha_editor"].Email,
				Name:  "Duplicate Invitee",
				Role:  "editor",
			},
			expectedStatus:   http.StatusConflict,
			expectedErrorKey: "emailConflict",
		},
		{
			name:         "error for unauthorized request",
			expectUnauth: true,
			requestBody: handlers.InviteUserRequest{
				Email: "unauth_invitee@example.com",
				Name:  "Unauth Invitee",
				Role:  "editor",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "validation error: missing email",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid email format",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "invalid-email", Name: "Test Name", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing name",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: name with only whitespace",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "whitespace@example.com", Name: "   ", Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: missing role",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error: invalid role value",
			loginUserKey:   "alpha_admin",
			requestBody:    handlers.InviteUserRequest{Email: "valid@example.com", Name: "Test Name", Role: "guest"},
			expectedStatus: http.StatusBadRequest,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err, "Failed to marshal request body for test: %s", tt.name)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/users/invite", setup.BaseURL), bytes.NewBuffer(payloadBytes))
			assert.NoError(t, err, "Failed to create request for test: %s", tt.name)
			req.Header.Set("Content-Type", "application/json")

			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key not found in setup.TestUsersData: %s for test: %s", tt.loginUserKey, tt.name)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request for test: %s", tt.name)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s. Body: %s - expected: %d, actual: %d", tt.name, string(payloadBytes), tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusConflict && tt.expectedErrorKey != "" {
				var errResp ErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&errResp)
				assert.NoError(t, err, "Failed to decode error response for test: %s", tt.name)

				expectedMsg, msgOk := expectedInviteErrorMessages[tt.expectedErrorKey]
				assert.True(t, msgOk, "Expected error key %s not found in error messages map for test: %s", tt.expectedErrorKey, tt.name)
				assert.Equal(t, expectedMsg, errResp.Error, "Error message mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestAcceptInvite_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)

	const (
		plainValidTokenForAccept       = "26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ"
		plainNonExistentTokenForAccept = "accept-invite-plain-nonexistent-token-for-testing-456"
		plainExpiredTokenForAccept     = "accept-invite-plain-expired-token-for-testing-789"
		plainTokenForActiveUserAccept  = "accept-invite-plain-token-for-active-user-000"
		newPasswordForAcceptInvite     = "SuP3rS3cur3N3wP@sswOrd!"
	)

	type acceptInviteTestCase struct {
		name           string
		requestBody    handlers.AcceptInviteRequest
		expectedStatus int
	}

	tests := []acceptInviteTestCase{
		{
			name: "successful invite acceptance",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "token not found",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainNonExistentTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - password mismatch",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept, // Needs a valid token context for this to be the failure point
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: "IncorrectP@sswOrdConfirm",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - password too short",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - empty token",
			requestBody: handlers.AcceptInviteRequest{
				Token:           "",
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "expired token",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainExpiredTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user status not pending_verification (e.g., already active)",
			requestBody: handlers.AcceptInviteRequest{
				Token:           plainTokenForActiveUserAccept, // Token associated with an already active user
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.requestBody)
			assert.NoError(t, err, "Test: %s, Failed to marshal request body", tt.name)

			reqURL := fmt.Sprintf("%s/auth/accept-invite", setup.BaseURL)
			req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(payloadBytes))
			assert.NoError(t, err, "Test: %s, Failed to create request", tt.name)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			assert.NoError(t, err, "Test: %s, Failed to execute request", tt.name)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			bodyBytes, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Test: %s, Expected status %d, got %d. Body: %s", tt.name, tt.expectedStatus, resp.StatusCode, string(bodyBytes))
		})
	}
}

func extractInvitationTokenFromEmail(t *testing.T, email *SendgridMockEmail) (string, error) {
	t.Helper()
	for _, content := range email.Content {
		if content.Type == "text/html" {
			re := regexp.MustCompile(`auth/accept-invite\?token=([^&"'>\s]+)`)
			matches := re.FindStringSubmatch(content.Value)
			if len(matches) > 1 {
				return matches[1], nil
			}
		}
	}
	return "", fmt.Errorf("invitation token not found in email HTML content")
}

func TestResendInvite_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}
	ctx := context.Background()

	initialPlainTokenForPendingUser := "accept-invite-plain-valid-token-for-testing-123"
	initialHashBytes := sha256.Sum256([]byte(initialPlainTokenForPendingUser))
	initialTokenHashForPendingUser := hex.EncodeToString(initialHashBytes[:])

	tests := []struct {
		name                 string
		loginUserKey         string
		targetUserKey        string
		targetUserIDOverride string
		expectedStatus       int // For single-call tests, or the *second* call in a rate-limit test if not handled by customAssertions
		expectUnauth         bool
		expectForbidden      bool
		customAssertions     func(t *testing.T, resp *http.Response, invokerUser testUserDetail, targetUser testUserDetail, firstCallRespStatus int)
		isRateLimitTest      bool // Indicates if this is the multi-call rate limit test
	}{
		{
			name:           "successful resend by admin for pending user (pending_editor_valid_token)",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "pending_editor_valid_token",
			expectedStatus: http.StatusOK,
			customAssertions: func(t *testing.T, resp *http.Response, invokerUser testUserDetail, targetUser testUserDetail, firstCallRespStatus int) {
				assert.Equal(t, http.StatusOK, resp.StatusCode, "Response status should be OK")

				var r SuccessResponse
				err := json.NewDecoder(resp.Body).Decode(&r)
				assert.NoError(t, err)
				assert.True(t, r.Success)

				var countOldToken int
				err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM invitation_tokens WHERE token_hash = $1 AND user_id = $2", initialTokenHashForPendingUser, targetUser.UserID).Scan(&countOldToken)
				assert.NoError(t, err, "DB query for old token count failed")
				assert.Equal(t, 0, countOldToken, "Old token for pending_editor_valid_token should be deleted from DB")

				email, err := getLatestEmailFromSendgridMock(t, targetUser.Email)
				assert.NoError(t, err, "Failed to get email from SendGrid mock")

				var newPlainToken string
				if err == nil {
					assert.Contains(t, email.Personalizations[0].Subject, fmt.Sprintf("%sさんから%s様へのdislyzeへのご招待", invokerUser.Name, targetUser.Name))
					var errToken error
					newPlainToken, errToken = extractInvitationTokenFromEmail(t, email)
					assert.NoError(t, errToken, "Failed to extract new token from email")
					assert.NotEmpty(t, newPlainToken, "New plain token should not be empty")

					newTokenHashBytes := sha256.Sum256([]byte(newPlainToken))
					newTokenHash := hex.EncodeToString(newTokenHashBytes[:])

					var dbTokenHash string
					var dbUserID pgtype.UUID
					var dbExpiresAt pgtype.Timestamptz

					err = pool.QueryRow(ctx, "SELECT token_hash, user_id, expires_at FROM invitation_tokens WHERE token_hash = $1 AND user_id = $2", newTokenHash, targetUser.UserID).Scan(&dbTokenHash, &dbUserID, &dbExpiresAt)
					assert.NoError(t, err, "Failed to query new invitation token from DB")
					if err == nil {
						assert.Equal(t, newTokenHash, dbTokenHash)
						var expectedPgUUID pgtype.UUID
						scanErr := expectedPgUUID.Scan(targetUser.UserID)
						assert.NoError(t, scanErr)
						assert.Equal(t, expectedPgUUID, dbUserID)
						assert.True(t, dbExpiresAt.Time.After(time.Now()), "New token expiry should be in the future")
						assert.True(t, dbExpiresAt.Time.Before(time.Now().Add(49*time.Hour)), "New token expiry should be around 48 hours")
					}
				}
				if newPlainToken == "" {
					t.Fatal("newPlainToken was not extracted, cannot proceed with invite acceptance")
				}

				// --- Start: Accept Invite and Verify Activation ---
				const newPasswordForInviteAccept = "ValidNewPass123!"

				acceptInvitePayload := handlers.AcceptInviteRequest{
					Token:           newPlainToken,
					Password:        newPasswordForInviteAccept,
					PasswordConfirm: newPasswordForInviteAccept,
				}
				payloadBytes, err := json.Marshal(acceptInvitePayload)
				assert.NoError(t, err, "Failed to marshal AcceptInviteRequest")

				acceptInviteURL := fmt.Sprintf("%s/auth/accept-invite", setup.BaseURL)
				acceptInviteReq, err := http.NewRequest(http.MethodPost, acceptInviteURL, bytes.NewBuffer(payloadBytes))
				assert.NoError(t, err, "Failed to create AcceptInvite request")
				acceptInviteReq.Header.Set("Content-Type", "application/json")

				acceptInviteResp, err := client.Do(acceptInviteReq)
				assert.NoError(t, err, "Failed to execute AcceptInvite request")
				defer func() {
					if err := acceptInviteResp.Body.Close(); err != nil {
						t.Logf("Error closing acceptInviteResp body: %v", err)
					}
				}()

				acceptInviteBodyBytes, _ := io.ReadAll(acceptInviteResp.Body)
				assert.Equal(t, http.StatusOK, acceptInviteResp.StatusCode, "AcceptInvite request failed. Body: %s", string(acceptInviteBodyBytes))

				var acceptInviteSuccessResp map[string]bool
				err = json.Unmarshal(acceptInviteBodyBytes, &acceptInviteSuccessResp)
				assert.NoError(t, err, "Failed to decode AcceptInvite response")
				assert.True(t, acceptInviteSuccessResp["success"], "AcceptInvite response success field was false")

				var userStatus string
				err = pool.QueryRow(ctx, "SELECT status FROM users WHERE id = $1", targetUser.UserID).Scan(&userStatus)
				assert.NoError(t, err, "Failed to query user status after invite acceptance")
				assert.Equal(t, "active", userStatus, "User status should be active after invite acceptance")

				activatedUserAccessToken, _ := setup.LoginUserAndGetTokens(t, targetUser.Email, newPasswordForInviteAccept)
				assert.NotEmpty(t, activatedUserAccessToken, "Access token should not be empty after login for activated user")
				// --- End: Accept Invite and Verify Activation ---
			},
		},
		{
			name:                 "target user not found",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "00000000-0000-0000-0000-000000000000", // Non-existent UUID
			expectedStatus:       http.StatusInternalServerError,
		},
		{
			name:           "target user not pending - active",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor", // An active user
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "target user not pending - suspended",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "suspended_editor",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:            "invoker not admin",
			loginUserKey:    "alpha_editor",
			targetUserKey:   "pending_editor_valid_token",
			expectedStatus:  http.StatusForbidden,
			expectForbidden: true,
		},
		{
			name:           "unauthenticated request",
			targetUserKey:  "pending_editor_valid_token",
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:            "invoker and target in different tenants",
			loginUserKey:    "beta_admin",                                // Tenant B
			targetUserKey:   "pending_editor_tenant_A_for_x_tenant_test", // Tenant A
			expectedStatus:  http.StatusForbidden,
			expectForbidden: true,
		},
		{
			name:                 "invalid target user ID format",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "not-a-uuid",
			expectedStatus:       http.StatusInternalServerError,
		},
		{
			name:            "rate limit: first call OK, second call TooManyRequests",
			loginUserKey:    "alpha_admin",
			targetUserKey:   "pending_editor_for_rate_limit_test",
			isRateLimitTest: true,
			// For rate limit tests, expectedStatus in the struct is for the *second* call if not handled by customAssertions.
			// Here, customAssertions will handle all checks.
			expectedStatus: http.StatusTooManyRequests,
			customAssertions: func(t *testing.T, resp *http.Response, invokerUser testUserDetail, targetUser testUserDetail, firstCallRespStatus int) {
				assert.Equal(t, http.StatusOK, firstCallRespStatus, "First call for rate limit test should succeed (200 OK)")

				// Assertions for the second call (which is the `resp` passed to this function)
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Second call for rate limit test should be 429 Too Many Requests")

				if resp.StatusCode == http.StatusTooManyRequests {
					var errResp ErrorResponse
					bodyBytes, _ := io.ReadAll(resp.Body)
					errDecode := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&errResp)
					resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					assert.NoError(t, errDecode, "Failed to decode rate limit error response")
					assert.Equal(t, "招待メールの再送信は、ユーザーごとに5分間に1回のみ可能です。しばらくしてから再度お試しください。", errResp.Error, "Rate limit error message mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var invokerDetails testUserDetail
			var targetDetails testUserDetail
			var accessToken string

			if !tt.expectUnauth && tt.loginUserKey != "" {
				var ok bool
				invokerDataFromSetup, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key '%s' not found in setup.TestUsersData", tt.loginUserKey)
				invokerDetails = testUserDetail(invokerDataFromSetup)

				var invokerRefreshToken string
				accessToken, invokerRefreshToken = setup.LoginUserAndGetTokens(t, invokerDetails.Email, invokerDetails.PlainTextPassword)
				assert.NotEmpty(t, accessToken, "Access token for invoker %s should not be empty", invokerDetails.Email)
				assert.NotEmpty(t, invokerRefreshToken, "Refresh token for invoker %s should not be empty", invokerDetails.Email)
			}

			targetUserID := ""
			if tt.targetUserKey != "" {
				var ok bool
				targetDataFromSetup, ok := setup.TestUsersData[tt.targetUserKey]
				assert.True(t, ok, "Target user key '%s' not found in setup.TestUsersData", tt.targetUserKey)
				targetDetails = testUserDetail(targetDataFromSetup)
				targetUserID = targetDetails.UserID
			} else if tt.targetUserIDOverride != "" {
				targetUserID = tt.targetUserIDOverride
			}
			assert.NotEmpty(t, targetUserID, "Target User ID must be set for test '%s'", tt.name)

			var firstCallActualStatus int
			var finalResp *http.Response
			var finalBodyBytes []byte

			// --- First Call (for all tests) ---
			reqURL := fmt.Sprintf("%s/users/%s/resend-invite", setup.BaseURL, targetUserID)
			req, errConstruct := http.NewRequest("POST", reqURL, nil)
			assert.NoError(t, errConstruct)

			if !tt.expectUnauth && accessToken != "" {
				req.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})
			}

			firstResp, errDo := client.Do(req)
			assert.NoError(t, errDo)

			firstBodyBytes, errRead := io.ReadAll(firstResp.Body)
			assert.NoError(t, errRead)
			defer func() {
				if err := firstResp.Body.Close(); err != nil {
					t.Logf("Error closing firstResp body: %v", err)
				}
			}()
			firstResp.Body = io.NopCloser(bytes.NewBuffer(firstBodyBytes))
			firstCallActualStatus = firstResp.StatusCode

			finalResp = firstResp
			finalBodyBytes = firstBodyBytes

			// --- Second Call (only for rate limit test) ---
			if tt.isRateLimitTest {
				// Make the second call to the same endpoint
				secondReqURL := fmt.Sprintf("%s/users/%s/resend-invite", setup.BaseURL, targetUserID)
				secondReq, errConstructSecond := http.NewRequest("POST", secondReqURL, nil)
				assert.NoError(t, errConstructSecond)

				if !tt.expectUnauth && accessToken != "" {
					secondReq.AddCookie(&http.Cookie{Name: "dislyze_access_token", Value: accessToken, Path: "/"})
				}

				secondResp, errDoSecond := client.Do(secondReq)
				assert.NoError(t, errDoSecond)

				secondBodyBytes, errReadSecond := io.ReadAll(secondResp.Body)
				assert.NoError(t, errReadSecond)
				defer func() {
					if err := secondResp.Body.Close(); err != nil {
						t.Logf("Error closing secondResp body: %v", err)
					}
				}()
				secondResp.Body = io.NopCloser(bytes.NewBuffer(secondBodyBytes))

				finalResp = secondResp
				finalBodyBytes = secondBodyBytes
			}

			// --- Assertions ---
			// For non-rate-limit tests, expectedStatus applies to the first (and only) call.
			// The customAssertions will handle specific status checks for rate-limit tests.
			if !tt.isRateLimitTest {
				assert.Equal(t, tt.expectedStatus, firstCallActualStatus, "Status code mismatch for test: %s. Body: %s", tt.name, string(finalBodyBytes))
			}

			if tt.customAssertions != nil {
				// Ensure the body of finalResp is ready to be read by customAssertions
				finalResp.Body = io.NopCloser(bytes.NewBuffer(finalBodyBytes))
				tt.customAssertions(t, finalResp, invokerDetails, targetDetails, firstCallActualStatus)
			} else if tt.isRateLimitTest && tt.expectedStatus == http.StatusTooManyRequests {
				// This is a fallback if a rate limit test somehow doesn't have custom assertions
				// but expects TooManyRequests on the second call.
				assert.Equal(t, http.StatusTooManyRequests, finalResp.StatusCode, "Expected 429 on second call for rate limit test: %s", tt.name)
				var errResp ErrorResponse
				finalResp.Body = io.NopCloser(bytes.NewBuffer(finalBodyBytes))
				errDecode := json.NewDecoder(finalResp.Body).Decode(&errResp)
				assert.NoError(t, errDecode, "Failed to decode rate limit error response for test: %s", tt.name)
				assert.Equal(t, "招待メールの再送信は、ユーザーごとに5分間に1回のみ可能です。しばらくしてから再度お試しください。", errResp.Error, "Rate limit error message mismatch for test: %s", tt.name)
			}

			if finalResp != nil && finalResp.Body != nil {
				defer func() {
					if err := finalResp.Body.Close(); err != nil {
						t.Logf("Error closing finalResp body: %v", err)
					}
				}()
			}
		})
	}
}

func TestGetMe_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                string
		loginUserKey        string // Key for setup.TestUsersData map
		expectedStatus      int
		expectedTenantName  string
		expectedTenantPlan  string
		expectedUserRole    string
		expectErrorResponse bool
	}{
		{
			name:               "alpha_admin gets their details",
			loginUserKey:       "alpha_admin",
			expectedStatus:     http.StatusOK,
			expectedTenantName: "Tenant Alpha",
			expectedTenantPlan: "basic",
			expectedUserRole:   "admin",
		},
		{
			name:               "alpha_editor gets their details",
			loginUserKey:       "alpha_editor",
			expectedStatus:     http.StatusOK,
			expectedTenantName: "Tenant Alpha",
			expectedTenantPlan: "basic",
			expectedUserRole:   "editor",
		},
		{
			name:                "unauthenticated user gets 401",
			loginUserKey:        "", // No login
			expectedStatus:      http.StatusUnauthorized,
			expectErrorResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var cookies []*http.Cookie
			var currentUserDetails setup.UserTestData

			if tt.loginUserKey != "" {
				var ok bool
				currentUserDetails, ok = setup.TestUsersData[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: User key '%s' not found in TestUsersData", tt.loginUserKey)
				}

				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, currentUserDetails.Email, currentUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			req, err := http.NewRequest("GET", fmt.Sprintf("%s/me", setup.BaseURL), nil)
			assert.NoError(t, err)

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

			if tt.expectedStatus == http.StatusOK {
				var meResponse handlers.MeResponse
				err = json.NewDecoder(resp.Body).Decode(&meResponse)
				assert.NoError(t, err, "Failed to decode MeResponse")

				assert.Equal(t, currentUserDetails.UserID, meResponse.UserID)
				assert.Equal(t, currentUserDetails.Email, meResponse.Email)
				assert.Equal(t, currentUserDetails.Name, meResponse.UserName)
				assert.Equal(t, tt.expectedTenantName, meResponse.TenantName)
				assert.Equal(t, tt.expectedTenantPlan, meResponse.TenantPlan)
				assert.Equal(t, tt.expectedUserRole, meResponse.UserRole)
			} else if tt.expectErrorResponse {
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received error response body for %s: %s", tt.name, string(bodyBytes)) // Log for debugging if needed
			}
		})
	}
}

func CheckUserExists(t *testing.T, pool *pgxpool.Pool, userID string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	assert.NoError(t, err, "Error checking if user exists")
	return exists
}

func CheckInvitationTokensExistForUser(t *testing.T, pool *pgxpool.Pool, userID string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM invitation_tokens WHERE user_id = $1)", userID).Scan(&exists)
	assert.NoError(t, err, "Error checking if invitation tokens exist for user")
	return exists
}

func CheckRefreshTokensExistForUser(t *testing.T, pool *pgxpool.Pool, userID string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1)", userID).Scan(&exists)
	assert.NoError(t, err, "Error checking if refresh tokens exist for user")
	return exists
}

func CheckPasswordResetTokensExistForUser(t *testing.T, pool *pgxpool.Pool, userID string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM password_reset_tokens WHERE user_id = $1)", userID).Scan(&exists)
	assert.NoError(t, err, "Error checking if password reset tokens exist for user")
	return exists
}

func TestDeleteUser_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name              string
		loginUserKey      string // Key for setup.TestUsersData of the user performing the delete
		targetUserKey     string // Key for setup.TestUsersData of the user to be deleted (can be same as loginUserKey or different)
		targetUserIDInput string // Used if targetUserKey is empty (e.g. non-existent user, invalid format)
		expectedStatus    int
		expectedErrorMsg  string
		preTestSetup      func(t *testing.T) // Optional setup function to run before the test, e.g. to reset DB state
	}{
		{
			name:           "Admin Deletes Editor - Success",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:             "Admin Tries to Delete Self - Conflict",
			loginUserKey:     "alpha_admin",
			targetUserKey:    "alpha_admin",
			expectedStatus:   http.StatusConflict,
			expectedErrorMsg: "自分自身を削除することはできません。",
		},
		{
			name:           "Admin Tries to Delete User in Another Tenant - Forbidden",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "beta_admin", // beta_admin is in a different tenant
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Editor Tries to Delete User - Forbidden",
			loginUserKey:   "alpha_editor",
			targetUserKey:  "pending_editor_valid_token",
			expectedStatus: http.StatusForbidden, // Middleware RequireAdmin should block this
			preTestSetup: func(t *testing.T) {
				// Reset DB because alpha_editor was deleted in a prior test case
				setup.CleanupDB(t, pool)
				setup.SeedDB(t, pool)
			},
		},
		{
			name:              "Delete Non-Existent User - NotFound",
			loginUserKey:      "alpha_admin",
			targetUserIDInput: "00000000-0000-0000-0000-000000000000", // A valid UUID that won't exist
			expectedStatus:    http.StatusNotFound,
		},
		{
			name:           "Unauthenticated Delete Attempt - Unauthorized",
			loginUserKey:   "", // No login
			targetUserKey:  "alpha_editor",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:              "Invalid UserID Format in URL - BadRequest",
			loginUserKey:      "alpha_admin",
			targetUserIDInput: "not-a-uuid",
			expectedStatus:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preTestSetup != nil {
				tt.preTestSetup(t)
			}

			var cookies []*http.Cookie
			var targetUserID string

			if tt.loginUserKey != "" {
				loginUserDetails, ok := setup.TestUsersData[tt.loginUserKey]
				if !ok {
					t.Fatalf("Test setup error: Login user key '%s' not found in TestUsersData", tt.loginUserKey)
				}
				accessToken, refreshToken := setup.LoginUserAndGetTokens(t, loginUserDetails.Email, loginUserDetails.PlainTextPassword)
				cookies = []*http.Cookie{
					{Name: "dislyze_access_token", Value: accessToken},
					{Name: "dislyze_refresh_token", Value: refreshToken},
				}
			}

			if tt.targetUserKey != "" {
				targetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				if !ok {
					t.Fatalf("Test setup error: Target user key '%s' not found in TestUsersData", tt.targetUserKey)
				}
				targetUserID = targetUserDetails.UserID
			} else {
				targetUserID = tt.targetUserIDInput
			}

			reqURL := fmt.Sprintf("%s/users/%s", setup.BaseURL, targetUserID)
			req, err := http.NewRequest("DELETE", reqURL, nil)
			assert.NoError(t, err)

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

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			if tt.expectedStatus == http.StatusNoContent {
				// Verify user is actually deleted from DB
				assert.False(t, CheckUserExists(t, pool, targetUserID), "User %s should have been deleted from DB", targetUserID)
				// Verify associated tokens are deleted
				assert.False(t, CheckInvitationTokensExistForUser(t, pool, targetUserID), "Invitation tokens for user %s should have been deleted", targetUserID)
				assert.False(t, CheckRefreshTokensExistForUser(t, pool, targetUserID), "Refresh tokens for user %s should have been deleted", targetUserID)
				// Password reset tokens are deleted by ON DELETE CASCADE with user, but good to be explicit if we had a direct call
			} else if tt.expectedErrorMsg != "" {
				var errResp ErrorResponse
				err = json.NewDecoder(resp.Body).Decode(&errResp)
				assert.NoError(t, err, "Failed to decode error response for test: %s", tt.name)
				assert.Equal(t, tt.expectedErrorMsg, errResp.Error, "Unexpected error message for test: %s", tt.name)
			} else if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK { // Log body for unexpected errors
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received unexpected error response body for %s (Status %d): %s", tt.name, resp.StatusCode, string(bodyBytes))
			}

			// For tests where user should NOT be deleted, verify they still exist
			if tt.expectedStatus != http.StatusNoContent && tt.targetUserKey != "" {
				originalTargetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				if ok {
					assert.True(t, CheckUserExists(t, pool, originalTargetUserDetails.UserID), "User %s should still exist in DB for test: %s", originalTargetUserDetails.UserID, tt.name)
				}
			}
		})
	}
}

func TestUpdateUserPermissions_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	client := &http.Client{}

	tests := []struct {
		name                 string
		loginUserKey         string // Key for setup.TestUsersData map, empty for unauth
		targetUserKey        string // Key for setup.TestUsersData map of target user
		targetUserIDOverride string // Use this if targetUserKey is empty (for invalid userID tests)
		requestBody          handlers.UpdateUserRoleRequest
		expectedStatus       int
		expectUnauth         bool
		validateResponse     func(t *testing.T, resp *http.Response) // For custom response validation
	}{
		// Authentication & Authorization Tests
		{
			name:           "unauthenticated request gets 401",
			targetUserKey:  "alpha_editor",
			requestBody:    handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus: http.StatusUnauthorized,
			expectUnauth:   true,
		},
		{
			name:           "non-admin user gets 403 forbidden",
			loginUserKey:   "alpha_editor",
			targetUserKey:  "pending_editor_valid_token",
			requestBody:    handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin from different tenant gets 403 forbidden",
			loginUserKey:   "beta_admin",   // Tenant B
			targetUserKey:  "alpha_editor", // Tenant A
			requestBody:    handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus: http.StatusForbidden,
		},

		// Input Validation Tests
		{
			name:                 "invalid userID format gets 400",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "not-a-uuid",
			requestBody:          handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus:       http.StatusBadRequest,
		},
		{
			name:           "empty role gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    handlers.UpdateUserRoleRequest{Role: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role value gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    handlers.UpdateUserRoleRequest{Role: "guest"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON request gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			expectedStatus: http.StatusBadRequest,
		},

		// Business Logic Tests
		{
			name:                 "non-existent user gets 404",
			loginUserKey:         "alpha_admin",
			targetUserIDOverride: "00000000-0000-0000-0000-000000000000", // Valid UUID that doesn't exist
			requestBody:          handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus:       http.StatusNotFound,
		},
		{
			name:           "user trying to update own role gets 400",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_admin", // Same user
			requestBody:    handlers.UpdateUserRoleRequest{Role: "editor"},
			expectedStatus: http.StatusBadRequest,
		},

		// Success Tests
		{
			name:           "admin successfully updates editor to admin",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor",
			requestBody:    handlers.UpdateUserRoleRequest{Role: "admin"},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
				var successResp map[string]bool
				err := json.NewDecoder(resp.Body).Decode(&successResp)
				assert.NoError(t, err, "Failed to decode success response")
				assert.True(t, successResp["success"], "Expected success to be true")
			},
		},
		{
			name:           "admin successfully updates admin to editor",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "alpha_editor", // Was updated to admin in previous test
			requestBody:    handlers.UpdateUserRoleRequest{Role: "editor"},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp *http.Response) {
				var successResp map[string]bool
				err := json.NewDecoder(resp.Body).Decode(&successResp)
				assert.NoError(t, err, "Failed to decode success response")
				assert.True(t, successResp["success"], "Expected success to be true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetUserID string
			if tt.targetUserKey != "" {
				targetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				assert.True(t, ok, "Target user key '%s' not found in setup.TestUsersData", tt.targetUserKey)
				targetUserID = targetUserDetails.UserID
			} else if tt.targetUserIDOverride != "" {
				targetUserID = tt.targetUserIDOverride
			} else {
				t.Fatal("Either targetUserKey or targetUserIDOverride must be provided")
			}

			var reqBody []byte
			var err error

			if tt.name == "malformed JSON request gets 400" {
				// Send malformed JSON for this specific test
				reqBody = []byte(`{"role": "admin", invalid}`)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err, "Failed to marshal request body")
			}

			reqURL := fmt.Sprintf("%s/users/%s/permissions", setup.BaseURL, targetUserID)
			req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqBody))
			assert.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Add authentication if not testing unauthenticated scenario
			if !tt.expectUnauth && tt.loginUserKey != "" {
				loginDetails, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key '%s' not found in setup.TestUsersData", tt.loginUserKey)

				accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)
				req.AddCookie(&http.Cookie{
					Name:  "dislyze_access_token",
					Value: accessToken,
					Path:  "/",
				})
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "Failed to execute request")
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code for test: %s", tt.name)

			// Custom response validation if provided
			if tt.validateResponse != nil {
				tt.validateResponse(t, resp)
			}

			// For successful updates, verify the role was actually changed in database
			if tt.expectedStatus == http.StatusOK && tt.targetUserKey != "" {
				ctx := context.Background()
				var actualRole string
				err = pool.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", targetUserID).Scan(&actualRole)
				assert.NoError(t, err, "Failed to query updated user role from database")

				expectedRole := strings.TrimSpace(strings.ToLower(string(tt.requestBody.Role)))
				assert.Equal(t, expectedRole, actualRole, "Role was not updated correctly in database")
			}
		})
	}
}

func TestGetUsersPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name               string
		page               int
		limit              int
		expectedStatus     int
		expectedPage       int
		expectedLimit      int
		expectedTotal      int
		expectedTotalPages int
		expectedHasNext    bool
		expectedHasPrev    bool
		expectedUserCount  int
	}{
		{
			name:               "page 1 with limit 2 - first page",
			page:               1,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      2,
			expectedTotal:      6, // Total users in Tenant A
			expectedTotalPages: 3, // 6 users / 2 per page = 3 pages
			expectedHasNext:    true,
			expectedHasPrev:    false,
			expectedUserCount:  2,
		},
		{
			name:               "page 2 with limit 2 - middle page",
			page:               2,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       2,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    true,
			expectedHasPrev:    true,
			expectedUserCount:  2,
		},
		{
			name:               "page 3 with limit 2 - last page",
			page:               3,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       3,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    false,
			expectedHasPrev:    true,
			expectedUserCount:  2,
		},
		{
			name:               "page beyond total pages returns empty results",
			page:               5,
			limit:              2,
			expectedStatus:     http.StatusOK,
			expectedPage:       5,
			expectedLimit:      2,
			expectedTotal:      6,
			expectedTotalPages: 3,
			expectedHasNext:    false,
			expectedHasPrev:    true,
			expectedUserCount:  0,
		},
		{
			name:               "large limit gets all users in one page",
			page:               1,
			limit:              10,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      10,
			expectedTotal:      6,
			expectedTotalPages: 1,
			expectedHasNext:    false,
			expectedHasPrev:    false,
			expectedUserCount:  6,
		},
		{
			name:               "limit exceeding max (100) gets capped",
			page:               1,
			limit:              150,
			expectedStatus:     http.StatusOK,
			expectedPage:       1,
			expectedLimit:      100, // Should be capped at 100
			expectedTotal:      6,
			expectedTotalPages: 1,
			expectedHasNext:    false,
			expectedHasPrev:    false,
			expectedUserCount:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=%d&limit=%d", setup.BaseURL, tt.page, tt.limit)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse handlers.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedLimit, usersResponse.Pagination.Limit, "Limit mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotal, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotalPages, usersResponse.Pagination.TotalPages, "TotalPages mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasNext, usersResponse.Pagination.HasNext, "HasNext mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasPrev, usersResponse.Pagination.HasPrev, "HasPrev mismatch for test: %s", tt.name)

				// Verify user count
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestGetUsersSearch_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name                string
		search              string
		expectedStatus      int
		expectedUserCount   int
		expectedContains    []string // Emails that should be in results
		expectedNotContains []string // Emails that should not be in results
	}{
		{
			name:              "search by name 'Admin' finds admin users",
			search:            "Admin",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:              "search by name 'Editor' finds editor users",
			search:            "Editor",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 5, // alpha_editor, pending_editor_valid_token, suspended_editor, pending_editor_for_rate_limit_test, pending_editor_tenant_A_for_x_tenant_test
			expectedContains: []string{
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:              "search by partial name 'Pending' finds pending users",
			search:            "Pending",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 3, // All pending users
			expectedContains: []string{
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
			},
		},
		{
			name:              "search by email domain 'alpha' finds alpha users",
			search:            "alpha",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 2,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
			},
			expectedNotContains: []string{
				setup.TestUsersData["pending_editor_valid_token"].Email,
			},
		},
		{
			name:              "case insensitive search 'ADMIN' finds admin",
			search:            "ADMIN",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
			},
		},
		{
			name:              "search for 'Suspended' finds suspended user",
			search:            "Suspended",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 1,
			expectedContains: []string{
				setup.TestUsersData["suspended_editor"].Email,
			},
		},
		{
			name:              "search for nonexistent term returns empty results",
			search:            "nonexistent",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 0,
			expectedContains:  []string{},
		},
		{
			name:              "empty search returns all users",
			search:            "",
			expectedStatus:    http.StatusOK,
			expectedUserCount: 6, // All users in Tenant A
			expectedContains: []string{
				setup.TestUsersData["alpha_admin"].Email,
				setup.TestUsersData["alpha_editor"].Email,
				setup.TestUsersData["pending_editor_valid_token"].Email,
				setup.TestUsersData["suspended_editor"].Email,
				setup.TestUsersData["pending_editor_for_rate_limit_test"].Email,
				setup.TestUsersData["pending_editor_tenant_A_for_x_tenant_test"].Email,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=1&limit=50&search=%s", setup.BaseURL, tt.search)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse handlers.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify user count
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)

				// Verify total in pagination matches user count for these tests
				assert.Equal(t, tt.expectedUserCount, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)

				// Collect actual emails
				actualEmails := make([]string, len(usersResponse.Users))
				for i, user := range usersResponse.Users {
					actualEmails[i] = user.Email
				}

				// Verify expected emails are present
				for _, expectedEmail := range tt.expectedContains {
					assert.Contains(t, actualEmails, expectedEmail, "Expected email %s not found in results for test: %s", expectedEmail, tt.name)
				}

				// Verify unexpected emails are not present
				for _, unexpectedEmail := range tt.expectedNotContains {
					assert.NotContains(t, actualEmails, unexpectedEmail, "Unexpected email %s found in results for test: %s", unexpectedEmail, tt.name)
				}
			}
		})
	}
}

func TestGetUsersSearchWithPagination_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to 6 users in Tenant A
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name              string
		search            string
		page              int
		limit             int
		expectedStatus    int
		expectedTotal     int
		expectedUserCount int
		expectedPage      int
		expectedHasNext   bool
		expectedHasPrev   bool
	}{
		{
			name:              "search 'Editor' with pagination - page 1 limit 2",
			search:            "Editor",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     5, // 5 users with "Editor" in name
			expectedUserCount: 2, // First 2 results
			expectedPage:      1,
			expectedHasNext:   true,
			expectedHasPrev:   false,
		},
		{
			name:              "search 'Editor' with pagination - page 2 limit 2",
			search:            "Editor",
			page:              2,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     5,
			expectedUserCount: 2, // Next 2 results
			expectedPage:      2,
			expectedHasNext:   true, // Still more results (page 3 will have 1 result)
			expectedHasPrev:   true,
		},
		{
			name:              "search 'Admin' with pagination - single result",
			search:            "Admin",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     1, // Only 1 admin
			expectedUserCount: 1,
			expectedPage:      1,
			expectedHasNext:   false,
			expectedHasPrev:   false,
		},
		{
			name:              "search 'nonexistent' with pagination - no results",
			search:            "nonexistent",
			page:              1,
			limit:             2,
			expectedStatus:    http.StatusOK,
			expectedTotal:     0,
			expectedUserCount: 0,
			expectedPage:      1,
			expectedHasNext:   false,
			expectedHasPrev:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users?page=%d&limit=%d&search=%s", setup.BaseURL, tt.page, tt.limit, tt.search)
			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse handlers.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify pagination metadata
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTotal, usersResponse.Pagination.Total, "Total mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedUserCount, len(usersResponse.Users), "User count mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasNext, usersResponse.Pagination.HasNext, "HasNext mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedHasPrev, usersResponse.Pagination.HasPrev, "HasPrev mismatch for test: %s", tt.name)

				// Verify all returned users match the search term
				for _, user := range usersResponse.Users {
					nameMatch := strings.Contains(strings.ToLower(user.Name), strings.ToLower(tt.search))
					emailMatch := strings.Contains(strings.ToLower(user.Email), strings.ToLower(tt.search))
					if tt.search != "" && tt.search != "nonexistent" {
						assert.True(t, nameMatch || emailMatch,
							"User %s (%s) does not match search term '%s' for test: %s",
							user.Name, user.Email, tt.search, tt.name)
					}
				}
			}
		})
	}
}

func TestGetUsersInvalidParameters_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	setup.SeedDB(t, pool)
	defer setup.CloseDB(pool)

	// Use alpha_admin who has access to users
	loginDetails := setup.TestUsersData["alpha_admin"]
	accessToken, _ := setup.LoginUserAndGetTokens(t, loginDetails.Email, loginDetails.PlainTextPassword)

	client := &http.Client{}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedPage   int
		expectedLimit  int
	}{
		{
			name:           "invalid page parameter - non-numeric defaults to 1",
			queryParams:    "page=abc&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "invalid limit parameter - non-numeric defaults to 50",
			queryParams:    "page=1&limit=xyz",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "negative page parameter defaults to 1",
			queryParams:    "page=-1&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "zero page parameter defaults to 1",
			queryParams:    "page=0&limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
		{
			name:           "negative limit parameter defaults to 50",
			queryParams:    "page=1&limit=-5",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "zero limit parameter defaults to 50",
			queryParams:    "page=1&limit=0",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "missing parameters use defaults",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "only page parameter provided",
			queryParams:    "page=2",
			expectedStatus: http.StatusOK,
			expectedPage:   2,  // Valid page should be preserved
			expectedLimit:  50, // Should default to limit=50
		},
		{
			name:           "only limit parameter provided",
			queryParams:    "limit=10",
			expectedStatus: http.StatusOK,
			expectedPage:   1,  // Should default to page=1
			expectedLimit:  10, // Valid limit should be preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("%s/users", setup.BaseURL)
			if tt.queryParams != "" {
				reqURL += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", reqURL, nil)
			assert.NoError(t, err)

			req.AddCookie(&http.Cookie{
				Name:  "dislyze_access_token",
				Value: accessToken,
				Path:  "/",
			})

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch for test: %s", tt.name)

			if tt.expectedStatus == http.StatusOK {
				var usersResponse handlers.GetUsersResponse
				err = json.NewDecoder(resp.Body).Decode(&usersResponse)
				assert.NoError(t, err, "Failed to decode response for test: %s", tt.name)

				// Verify the exact expected default values are applied
				assert.Equal(t, tt.expectedPage, usersResponse.Pagination.Page, "Page should match expected default for test: %s", tt.name)
				assert.Equal(t, tt.expectedLimit, usersResponse.Pagination.Limit, "Limit should match expected default for test: %s", tt.name)

				// Additional validation to ensure reasonable values
				assert.True(t, usersResponse.Pagination.Page >= 1, "Page should be at least 1")
				assert.True(t, usersResponse.Pagination.Limit >= 1, "Limit should be at least 1")
				assert.True(t, usersResponse.Pagination.Limit <= 100, "Limit should not exceed 100")
			}
		})
	}
}
