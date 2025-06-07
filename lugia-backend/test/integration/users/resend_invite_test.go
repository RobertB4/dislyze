package users

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"lugia/features/auth"
	"lugia/test/integration/setup"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func extractInvitationTokenFromEmail(t *testing.T, email *setup.SendgridMockEmail) (string, error) {
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
	setup.ResetAndSeedDB(t, pool)
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
		customAssertions     func(t *testing.T, resp *http.Response, invokerUser setup.UserTestData, targetUser setup.UserTestData, firstCallRespStatus int)
		isRateLimitTest      bool // Indicates if this is the multi-call rate limit test
	}{
		{
			name:           "successful resend by admin for pending user (pending_editor_valid_token)",
			loginUserKey:   "alpha_admin",
			targetUserKey:  "pending_editor_valid_token",
			expectedStatus: http.StatusOK,
			customAssertions: func(t *testing.T, resp *http.Response, invokerUser setup.UserTestData, targetUser setup.UserTestData, firstCallRespStatus int) {
				assert.Equal(t, http.StatusOK, resp.StatusCode, "Response status should be OK")

				var countOldToken int
				err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM invitation_tokens WHERE token_hash = $1 AND user_id = $2", initialTokenHashForPendingUser, targetUser.UserID).Scan(&countOldToken)
				assert.NoError(t, err, "DB query for old token count failed")
				assert.Equal(t, 0, countOldToken, "Old token for pending_editor_valid_token should be deleted from DB")

				email, err := setup.GetLatestEmailFromSendgridMock(t, targetUser.Email)
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

				acceptInvitePayload := auth.AcceptInviteRequest{
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
			customAssertions: func(t *testing.T, resp *http.Response, invokerUser setup.UserTestData, targetUser setup.UserTestData, firstCallRespStatus int) {
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
			var invokerDetails setup.UserTestData
			var targetDetails setup.UserTestData
			var accessToken string

			if !tt.expectUnauth && tt.loginUserKey != "" {
				var ok bool
				invokerDataFromSetup, ok := setup.TestUsersData[tt.loginUserKey]
				assert.True(t, ok, "Login user key '%s' not found in setup.TestUsersData", tt.loginUserKey)
				invokerDetails = setup.UserTestData(invokerDataFromSetup)

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
				targetDetails = setup.UserTestData(targetDataFromSetup)
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
