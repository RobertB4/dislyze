package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangePasswordRequest_Validate(t *testing.T) {
	tests := []struct {
		name          string
		request       ChangePasswordRequest
		expectedError string
	}{
		{
			name: "valid request",
			request: ChangePasswordRequest{
				CurrentPassword:    "current123",
				NewPassword:        "newpassword123",
				NewPasswordConfirm: "newpassword123",
			},
			expectedError: "",
		},
		{
			name: "missing current password",
			request: ChangePasswordRequest{
				CurrentPassword:    "",
				NewPassword:        "newpassword123",
				NewPasswordConfirm: "newpassword123",
			},
			expectedError: "current password is required",
		},
		{
			name: "missing new password",
			request: ChangePasswordRequest{
				CurrentPassword:    "current123",
				NewPassword:        "",
				NewPasswordConfirm: "newpassword123",
			},
			expectedError: "new password is required",
		},
		{
			name: "new password too short",
			request: ChangePasswordRequest{
				CurrentPassword:    "current123",
				NewPassword:        "short",
				NewPasswordConfirm: "short",
			},
			expectedError: "new password must be at least 8 characters long",
		},
		{
			name: "passwords do not match",
			request: ChangePasswordRequest{
				CurrentPassword:    "current123",
				NewPassword:        "newpassword123",
				NewPasswordConfirm: "differentpassword123",
			},
			expectedError: "new passwords do not match",
		},
		{
			name: "current and new password are the same",
			request: ChangePasswordRequest{
				CurrentPassword:    "samepassword123",
				NewPassword:        "samepassword123",
				NewPasswordConfirm: "samepassword123",
			},
			expectedError: "new password must be different from current password",
		},
		{
			name: "whitespace trimming works",
			request: ChangePasswordRequest{
				CurrentPassword:    "  current123  ",
				NewPassword:        "  newpassword123  ",
				NewPasswordConfirm: "  newpassword123  ",
			},
			expectedError: "",
		},
		{
			name: "whitespace-only current password",
			request: ChangePasswordRequest{
				CurrentPassword:    "   ",
				NewPassword:        "newpassword123",
				NewPasswordConfirm: "newpassword123",
			},
			expectedError: "current password is required",
		},
		{
			name: "whitespace-only new password",
			request: ChangePasswordRequest{
				CurrentPassword:    "current123",
				NewPassword:        "   ",
				NewPasswordConfirm: "   ",
			},
			expectedError: "new password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}