package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lugia/test/integration/setup"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type AcceptInviteRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func TestAcceptInviteValidation(t *testing.T) {
	pool := setup.InitDB(t)
	setup.CleanupDB(t, pool)
	defer setup.CloseDB(pool)

	tests := []struct {
		name           string
		request        AcceptInviteRequest
		expectedStatus int
	}{
		{
			name: "missing token",
			request: AcceptInviteRequest{
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty token",
			request: AcceptInviteRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: AcceptInviteRequest{
				Token:           "some-token",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "short",
				PasswordConfirm: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "passwords do not match",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid token (valid format but doesn't exist)",
			request: AcceptInviteRequest{
				Token:           "invalid-but-valid-format-token",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			expectedStatus: http.StatusBadRequest, // Should be "招待リンクが無効か、期限切れです"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/accept-invite", setup.BaseURL), bytes.NewBuffer(body))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("Error closing response body: %v", err)
				}
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// For all these test cases, there should be no cookies set
			cookies := resp.Cookies()
			assert.Empty(t, cookies, "Expected no cookies for failed accept invite")
		})
	}
}