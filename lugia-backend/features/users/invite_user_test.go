package users

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInviteUserRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request InviteUserRequestBody
		wantErr bool
		errMsg  string
	}{
		// Valid Input Tests
		{
			name: "valid request with single role ID",
			request: InviteUserRequestBody{
				Email:   "test@example.com",
				Name:    "Test User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},
		{
			name: "valid request with multiple role IDs",
			request: InviteUserRequestBody{
				Email:   "multi@example.com",
				Name:    "Multi Role User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001", "e0000000-0000-0000-0000-000000000002"},
			},
			wantErr: false,
		},
		{
			name: "valid request with whitespace trimming",
			request: InviteUserRequestBody{
				Email:   "  test@example.com  ",
				Name:    "  Test User  ",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},

		// Email Validation Tests
		{
			name: "missing email (empty string)",
			request: InviteUserRequestBody{
				Email:   "",
				Name:    "Test User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "email with only whitespace",
			request: InviteUserRequestBody{
				Email:   "   ",
				Name:    "Test User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "invalid email format (no @)",
			request: InviteUserRequestBody{
				Email:   "invalid-email",
				Name:    "Test User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "email is invalid",
		},
		{
			name: "valid email format",
			request: InviteUserRequestBody{
				Email:   "valid@domain.com",
				Name:    "Test User",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},

		// Name Validation Tests
		{
			name: "missing name (empty string)",
			request: InviteUserRequestBody{
				Email:   "test@example.com",
				Name:    "",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "name is required and cannot be only whitespace",
		},
		{
			name: "name with only whitespace",
			request: InviteUserRequestBody{
				Email:   "test@example.com",
				Name:    "   ",
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "name is required and cannot be only whitespace",
		},

		// Role IDs Validation Tests
		{
			name: "missing role_ids (empty slice)",
			request: InviteUserRequestBody{
				Email:   "test@example.com",
				Name:    "Test User",
				RoleIDs: []string{},
			},
			wantErr: true,
			errMsg:  "at least one role is required",
		},
		{
			name: "missing role_ids (nil slice)",
			request: InviteUserRequestBody{
				Email:   "test@example.com",
				Name:    "Test User",
				RoleIDs: nil,
			},
			wantErr: true,
			errMsg:  "at least one role is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store original values to verify trimming
			originalEmail := tt.request.Email
			originalName := tt.request.Name
			originalRoleIDs := make([]string, len(tt.request.RoleIDs))
			copy(originalRoleIDs, tt.request.RoleIDs)

			err := tt.request.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
				
				// Verify trimming behavior for valid requests
				if tt.name == "valid request with whitespace trimming" {
					assert.Equal(t, "test@example.com", tt.request.Email, "Email should be trimmed")
					assert.Equal(t, "Test User", tt.request.Name, "Name should be trimmed")
				}
				
				// Verify RoleIDs are not modified
				assert.Equal(t, originalRoleIDs, tt.request.RoleIDs, "RoleIDs should not be modified by validation")
			}

			// Verify that email and name are always trimmed (even on error)
			assert.Equal(t, strings.TrimSpace(originalEmail), tt.request.Email, "Email should always be trimmed")
			assert.Equal(t, strings.TrimSpace(originalName), tt.request.Name, "Name should always be trimmed")
		})
	}
}
