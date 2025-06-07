package users

import (
	"testing"

	"lugia/queries_pregeneration"

	"github.com/stretchr/testify/assert"
)

func TestInviteUserRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request InviteUserRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: false,
		},
		{
			name: "valid request with editor role",
			request: InviteUserRequest{
				Email: "editor@example.com",
				Name:  "Editor User",
				Role:  queries_pregeneration.UserRole("editor"),
			},
			wantErr: false,
		},
		{
			name: "valid request with whitespace trimming",
			request: InviteUserRequest{
				Email: "  test@example.com  ",
				Name:  "  Test User  ",
				Role:  queries_pregeneration.UserRole("  ADMIN  "),
			},
			wantErr: false,
		},
		{
			name: "missing email",
			request: InviteUserRequest{
				Email: "",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "email with only whitespace",
			request: InviteUserRequest{
				Email: "   ",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "invalid email format",
			request: InviteUserRequest{
				Email: "invalid-email",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: true,
			errMsg:  "email is invalid",
		},
		{
			name: "missing name",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: true,
			errMsg:  "name is required and cannot be only whitespace",
		},
		{
			name: "name with only whitespace",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "   ",
				Role:  queries_pregeneration.UserRole("admin"),
			},
			wantErr: true,
			errMsg:  "name is required and cannot be only whitespace",
		},
		{
			name: "missing role",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole(""),
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "role with only whitespace",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("   "),
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "invalid role",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("guest"),
			},
			wantErr: true,
			errMsg:  "role is invalid, must be 'admin' or 'editor'",
		},
		{
			name: "another invalid role",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  queries_pregeneration.UserRole("superuser"),
			},
			wantErr: true,
			errMsg:  "role is invalid, must be 'admin' or 'editor'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
				// For valid requests, check that fields were properly trimmed and normalized
				if tt.name == "valid request with whitespace trimming" {
					assert.Equal(t, "test@example.com", tt.request.Email)
					assert.Equal(t, "Test User", tt.request.Name)
					assert.Equal(t, queries_pregeneration.UserRole("admin"), tt.request.Role)
				}
				// Ensure role is valid for all successful validations
				if tt.request.Role != "" {
					assert.True(t, tt.request.Role == "admin" || tt.request.Role == "editor")
				}
			}
		})
	}
}