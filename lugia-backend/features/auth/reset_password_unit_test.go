package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResetPasswordRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request ResetPasswordRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: nil,
		},
		{
			name: "missing token",
			request: ResetPasswordRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "missing password",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password too short",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "short",
				PasswordConfirm: "short",
			},
			wantErr: fmt.Errorf("password must be at least 8 characters long"),
		},
		{
			name: "passwords do not match",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			wantErr: fmt.Errorf("passwords do not match"),
		},
		{
			name: "empty token",
			request: ResetPasswordRequest{
				Token:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "whitespace-only token",
			request: ResetPasswordRequest{
				Token:           "   ",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "empty password",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "",
				PasswordConfirm: "",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "whitespace-only password",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "   ",
				PasswordConfirm: "   ",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password exactly 8 characters",
			request: ResetPasswordRequest{
				Token:           "valid-token-123",
				Password:        "12345678",
				PasswordConfirm: "12345678",
			},
			wantErr: nil,
		},
		{
			name: "fields with leading/trailing whitespace",
			request: ResetPasswordRequest{
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