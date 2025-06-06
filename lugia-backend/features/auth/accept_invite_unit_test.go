package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcceptInviteRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request AcceptInviteRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: AcceptInviteRequest{
				Token:           "valid-token-123",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: nil,
		},
		{
			name: "missing token",
			request: AcceptInviteRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "empty token",
			request: AcceptInviteRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "missing password",
			request: AcceptInviteRequest{
				Token:           "some-token",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password too short",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "short",
				PasswordConfirm: "short",
			},
			wantErr: fmt.Errorf("password must be at least 8 characters long"),
		},
		{
			name: "passwords do not match",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			wantErr: fmt.Errorf("passwords do not match"),
		},
		{
			name: "whitespace-only token",
			request: AcceptInviteRequest{
				Token:           "   ",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "empty password",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "",
				PasswordConfirm: "",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "whitespace-only password",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "   ",
				PasswordConfirm: "   ",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password exactly 8 characters",
			request: AcceptInviteRequest{
				Token:           "some-token",
				Password:        "12345678",
				PasswordConfirm: "12345678",
			},
			wantErr: nil,
		},
		{
			name: "fields with leading/trailing whitespace",
			request: AcceptInviteRequest{
				Token:           "  valid-token-123  ",
				Password:        "  password123  ",
				PasswordConfirm: "  password123  ",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}