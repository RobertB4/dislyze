package users

import (
	"testing"

	"lugia/queries_pregeneration"

	"github.com/stretchr/testify/assert"
)

func TestUpdateUserRoleRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateUserRoleRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid admin role",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("admin"),
			},
			wantErr: false,
		},
		{
			name: "valid editor role",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("editor"),
			},
			wantErr: false,
		},
		{
			name: "role with whitespace and case normalization",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("  ADMIN  "),
			},
			wantErr: false,
		},
		{
			name: "role with mixed case",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("Editor"),
			},
			wantErr: false,
		},
		{
			name: "missing role",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole(""),
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "role with only whitespace",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("   "),
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "invalid role value",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("guest"),
			},
			wantErr: true,
			errMsg:  "invalid role specified, must be 'admin' or 'editor'",
		},
		{
			name: "another invalid role value",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("superuser"),
			},
			wantErr: true,
			errMsg:  "invalid role specified, must be 'admin' or 'editor'",
		},
		{
			name: "invalid role with mixed case",
			request: UpdateUserRoleRequest{
				Role: queries_pregeneration.UserRole("GUEST"),
			},
			wantErr: true,
			errMsg:  "invalid role specified, must be 'admin' or 'editor'",
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
				// For successful validations, check that role was properly normalized
				assert.True(t, tt.request.Role == "admin" || tt.request.Role == "editor")
				// Check specific normalization cases
				if tt.name == "role with whitespace and case normalization" {
					assert.Equal(t, queries_pregeneration.UserRole("admin"), tt.request.Role)
				}
				if tt.name == "role with mixed case" {
					assert.Equal(t, queries_pregeneration.UserRole("editor"), tt.request.Role)
				}
			}
		})
	}
}