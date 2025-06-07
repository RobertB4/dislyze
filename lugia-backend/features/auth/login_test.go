package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequestBody
		wantErr error
	}{
		{
			name: "valid request",
			request: LoginRequestBody{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: nil,
		},
		{
			name: "missing email",
			request: LoginRequestBody{
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "missing password",
			request: LoginRequestBody{
				Email: "test@example.com",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		// Edge cases
		{
			name: "empty email",
			request: LoginRequestBody{
				Email:    "",
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "whitespace-only email",
			request: LoginRequestBody{
				Email:    "   ",
				Password: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "empty password",
			request: LoginRequestBody{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "whitespace-only password",
			request: LoginRequestBody{
				Email:    "test@example.com",
				Password: "   ",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "fields with leading/trailing whitespace",
			request: LoginRequestBody{
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
