package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignupRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request SignupRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: nil,
		},
		{
			name: "missing company name",
			request: SignupRequest{
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("company name is required"),
		},
		{
			name: "missing user name",
			request: SignupRequest{
				CompanyName:     "Test Company",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("user name is required"),
		},
		{
			name: "missing email",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "missing password",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password too short",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "short",
				PasswordConfirm: "short",
			},
			wantErr: fmt.Errorf("password must be at least 8 characters long"),
		},
		{
			name: "passwords do not match",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "different",
			},
			wantErr: fmt.Errorf("passwords do not match"),
		},
		// Edge cases
		{
			name: "empty company name",
			request: SignupRequest{
				CompanyName:     "",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("company name is required"),
		},
		{
			name: "whitespace-only company name",
			request: SignupRequest{
				CompanyName:     "   ",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("company name is required"),
		},
		{
			name: "empty user name",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("user name is required"),
		},
		{
			name: "whitespace-only user name",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "   ",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("user name is required"),
		},
		{
			name: "empty email",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "whitespace-only email",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "   ",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "empty password",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "",
				PasswordConfirm: "",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "whitespace-only password",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "   ",
				PasswordConfirm: "   ",
			},
			wantErr: fmt.Errorf("password is required"),
		},
		{
			name: "password exactly 8 characters",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "12345678",
				PasswordConfirm: "12345678",
			},
			wantErr: nil,
		},
		{
			name: "password with spaces",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "password 123",
				PasswordConfirm: "password 123",
			},
			wantErr: nil,
		},
		{
			name: "password with special characters",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				Password:        "p@ssw0rd!",
				PasswordConfirm: "p@ssw0rd!",
			},
			wantErr: nil,
		},
		{
			name: "fields with leading/trailing whitespace",
			request: SignupRequest{
				CompanyName:     "  Test Company  ",
				UserName:        "  Test User  ",
				Email:           "  test@example.com  ",
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