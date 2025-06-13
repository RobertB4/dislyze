package users

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

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
	setup.ResetAndSeedDB(t, pool)
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
			loginUserKey:   "enterprise_1",
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusOK,
		},
		{
			name:             "Admin Tries to Delete Self - Conflict",
			loginUserKey:     "enterprise_1",
			targetUserKey:    "enterprise_1",
			expectedStatus:   http.StatusConflict,
			expectedErrorMsg: "自分自身を削除することはできません。",
		},
		{
			name:           "Admin Tries to Delete User in Another Tenant - Forbidden",
			loginUserKey:   "enterprise_1",
			targetUserKey:  "smb_1", // smb_1 is in a different tenant
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Editor Tries to Delete User - Forbidden",
			loginUserKey:   "enterprise_2",
			targetUserKey:  "enterprise_3",
			expectedStatus: http.StatusForbidden, // Middleware RequireAdmin should block this
			preTestSetup: func(t *testing.T) {
				// Reset DB because enterprise_2 was deleted in a prior test case
				setup.ResetAndSeedDB(t, pool)
			},
		},
		{
			name:              "Delete Non-Existent User - NotFound",
			loginUserKey:      "enterprise_1",
			targetUserIDInput: "00000000-0000-0000-0000-000000000000", // A valid UUID that won't exist
			expectedStatus:    http.StatusNotFound,
		},
		{
			name:           "Unauthenticated Delete Attempt - Unauthorized",
			loginUserKey:   "", // No login
			targetUserKey:  "enterprise_2",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:              "Invalid UserID Format in URL - BadRequest",
			loginUserKey:      "enterprise_1",
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

			reqURL := fmt.Sprintf("%s/users/%s/delete", setup.BaseURL, targetUserID)
			req, err := http.NewRequest("POST", reqURL, nil)
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

			if tt.expectedStatus == http.StatusOK {
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
			} else if resp.StatusCode != http.StatusOK { // Log body for unexpected errors
				bodyBytes, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				t.Logf("Received unexpected error response body for %s (Status %d): %s", tt.name, resp.StatusCode, string(bodyBytes))
			}

			// For tests where user should NOT be deleted, verify they still exist
			if tt.expectedStatus != http.StatusOK && tt.targetUserKey != "" {
				originalTargetUserDetails, ok := setup.TestUsersData[tt.targetUserKey]
				if ok {
					assert.True(t, CheckUserExists(t, pool, originalTargetUserDetails.UserID), "User %s should still exist in DB for test: %s", originalTargetUserDetails.UserID, tt.name)
				}
			}
		})
	}
}
