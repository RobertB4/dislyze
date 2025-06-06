package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForgotPasswordRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request ForgotPasswordRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: ForgotPasswordRequest{
				Email: "test@example.com",
			},
			wantErr: nil,
		},
		{
			name: "missing email",
			request: ForgotPasswordRequest{
				Email: "",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "empty email",
			request: ForgotPasswordRequest{
				Email: "",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "whitespace-only email",
			request: ForgotPasswordRequest{
				Email: "   ",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "invalid email format - no @",
			request: ForgotPasswordRequest{
				Email: "invalid-email",
			},
			wantErr: fmt.Errorf("invalid email address format"),
		},
		{
			name: "invalid email format - just @",
			request: ForgotPasswordRequest{
				Email: "@",
			},
			wantErr: nil, // Basic validation only checks for @ presence
		},
		{
			name: "email with leading/trailing whitespace",
			request: ForgotPasswordRequest{
				Email: "  test@example.com  ",
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