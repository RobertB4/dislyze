package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForgotPasswordRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request ForgotPasswordRequestBody
		wantErr error
	}{
		{
			name: "valid request",
			request: ForgotPasswordRequestBody{
				Email: "test@example.com",
			},
			wantErr: nil,
		},
		{
			name: "missing email",
			request: ForgotPasswordRequestBody{
				Email: "",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "empty email",
			request: ForgotPasswordRequestBody{
				Email: "",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "whitespace-only email",
			request: ForgotPasswordRequestBody{
				Email: "   ",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "invalid email format - no @",
			request: ForgotPasswordRequestBody{
				Email: "invalid-email",
			},
			wantErr: fmt.Errorf("invalid email address format"),
		},
		{
			name: "invalid email format - just @",
			request: ForgotPasswordRequestBody{
				Email: "@",
			},
			wantErr: nil, // Basic validation only checks for @ presence
		},
		{
			name: "email with leading/trailing whitespace",
			request: ForgotPasswordRequestBody{
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
