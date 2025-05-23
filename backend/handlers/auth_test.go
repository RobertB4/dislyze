package handlers

import (
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
			wantErr: ErrCompanyNameRequired,
		},
		{
			name: "missing user name",
			request: SignupRequest{
				CompanyName:     "Test Company",
				Email:           "test@example.com",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: ErrUserNameRequired,
		},
		{
			name: "missing email",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: ErrEmailRequired,
		},
		{
			name: "missing password",
			request: SignupRequest{
				CompanyName:     "Test Company",
				UserName:        "Test User",
				Email:           "test@example.com",
				PasswordConfirm: "password123",
			},
			wantErr: ErrPasswordRequired,
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
			wantErr: ErrPasswordTooShort,
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
			wantErr: ErrPasswordsDoNotMatch,
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
			wantErr: ErrCompanyNameRequired,
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
			wantErr: ErrCompanyNameRequired,
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
			wantErr: ErrUserNameRequired,
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
			wantErr: ErrUserNameRequired,
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
			wantErr: ErrEmailRequired,
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
			wantErr: ErrEmailRequired,
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
			wantErr: ErrPasswordRequired,
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
			wantErr: ErrPasswordRequired,
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
			if err != tt.wantErr {
				t.Errorf("SignupRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
			wantErr: ErrEmailRequired,
		},
		{
			name: "missing password",
			request: LoginRequest{
				Email: "test@example.com",
			},
			wantErr: ErrPasswordRequired,
		},
		// Edge cases
		{
			name: "empty email",
			request: LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantErr: ErrEmailRequired,
		},
		{
			name: "whitespace-only email",
			request: LoginRequest{
				Email:    "   ",
				Password: "password123",
			},
			wantErr: ErrEmailRequired,
		},
		{
			name: "empty password",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: ErrPasswordRequired,
		},
		{
			name: "whitespace-only password",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "   ",
			},
			wantErr: ErrPasswordRequired,
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
			if err != tt.wantErr {
				t.Errorf("LoginRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAcceptInviteRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     AcceptInviteRequest
		wantErr bool
		errText string
	}{
		{
			name: "valid request",
			req: AcceptInviteRequest{
				Token:           "valid-token-string",
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: false,
		},
		{
			name: "empty token",
			req: AcceptInviteRequest{
				Token:           " ", // Test with whitespace
				Password:        "password123",
				PasswordConfirm: "password123",
			},
			wantErr: true,
			errText: "token is required",
		},
		{
			name: "empty password",
			req: AcceptInviteRequest{
				Token:           "valid-token-string",
				Password:        "",
				PasswordConfirm: "password123",
			},
			wantErr: true,
			errText: "password is required",
		},
		{
			name: "password too short",
			req: AcceptInviteRequest{
				Token:           "valid-token-string",
				Password:        "pass",
				PasswordConfirm: "pass",
			},
			wantErr: true,
			errText: "password must be at least 8 characters long",
		},
		{
			name: "passwords do not match",
			req: AcceptInviteRequest{
				Token:           "valid-token-string",
				Password:        "password123",
				PasswordConfirm: "password456",
			},
			wantErr: true,
			errText: "passwords do not match",
		},
		{
			name: "password confirm empty when password is not",
			req: AcceptInviteRequest{
				Token:           "valid-token-string",
				Password:        "password123",
				PasswordConfirm: "",
			},
			wantErr: true,
			errText: "passwords do not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.EqualError(t, err, tt.errText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
