package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: nil,
		},
		{
			name: "missing email",
			request: LoginRequest{
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "missing password",
			request: LoginRequest{
				Email: "test@example.com",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		// Edge cases
		{
			name: "empty email",
			request: LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "whitespace-only email",
			request: LoginRequest{
				Email:    "   ",
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "empty password",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "whitespace-only password",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "   ",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "fields with leading/trailing whitespace",
			request: LoginRequest{
				Email:    "  test@example.com  ",
				Password: "  password123  ",
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