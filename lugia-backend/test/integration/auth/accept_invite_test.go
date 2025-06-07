package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"lugia/features/auth"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcceptInvite_Integration(t *testing.T) {
	pool := setup.InitDB(t)
	defer setup.CloseDB(pool)

	setup.ResetAndSeedDB(t, pool)

	const (
		plainValidTokenForAccept       = "26U7PPxCPCFwWifs8gMD73Gq4tLIBlKBgroHOpkb1bQ"
		plainNonExistentTokenForAccept = "accept-invite-plain-nonexistent-token-for-testing-456"
		plainExpiredTokenForAccept     = "accept-invite-plain-expired-token-for-testing-789"
		plainTokenForActiveUserAccept  = "accept-invite-plain-token-for-active-user-000"
		newPasswordForAcceptInvite     = "SuP3rS3cur3N3wP@sswOrd!"
	)

	type acceptInviteTestCase struct {
		name           string
		requestBody    auth.AcceptInviteRequest
		expectedStatus int
	}

	tests := []acceptInviteTestCase{
		{
			name: "successful invite acceptance",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "token already used fails",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainValidTokenForAccept, // Same token as successful test
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "token not found",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainNonExistentTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - password mismatch",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainValidTokenForAccept, // Needs a valid token context for this to be the failure point
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: "IncorrectP@sswOrdConfirm",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - password too short",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainValidTokenForAccept,
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - empty token",
			requestBody: auth.AcceptInviteRequest{
				Token:           "",
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "expired token",
			requestBody: auth.AcceptInviteRequest{
				Token:           plainExpiredTokenForAccept,
				Password:        newPasswordForAcceptInvite,
				PasswordConfirm: newPasswordForAcceptInvite,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user status not pending_verification (e.g., already active)",
			requestBody: auth.AcceptInviteRequest{
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
