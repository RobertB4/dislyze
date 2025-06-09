package roles

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateRoleRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateRoleRequestBody
		wantErr bool
		errMsg  string
	}{
		// Valid Input Tests
		{
			name: "valid request with description",
			request: CreateRoleRequestBody{
				Name:          "Test Role",
				Description:   "A test role for testing",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},
		{
			name: "valid request with empty description",
			request: CreateRoleRequestBody{
				Name:          "Test Role",
				Description:   "",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},
		{
			name: "valid request with multiple permissions",
			request: CreateRoleRequestBody{
				Name:          "Multi Permission Role",
				Description:   "Role with multiple permissions",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001", "d0000000-0000-0000-0000-000000000002"},
			},
			wantErr: false,
		},
		{
			name: "valid request with whitespace trimming",
			request: CreateRoleRequestBody{
				Name:          "  Test Role  ",
				Description:   "  A test role  ",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},

		// Name Validation Tests
		{
			name: "missing name (empty string)",
			request: CreateRoleRequestBody{
				Name:          "",
				Description:   "Valid description",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing name (only whitespace)",
			request: CreateRoleRequestBody{
				Name:          "   ",
				Description:   "Valid description",
				PermissionIDs: []string{"d0000000-0000-0000-0000-000000000001"},
			},
			wantErr: true,
			errMsg:  "name is required",
		},

		// Permission Validation Tests
		{
			name: "missing permission IDs (empty array)",
			request: CreateRoleRequestBody{
				Name:          "Valid Role",
				Description:   "Valid description",
				PermissionIDs: []string{},
			},
			wantErr: true,
			errMsg:  "at least one permission is required",
		},
		{
			name: "missing permission IDs (nil)",
			request: CreateRoleRequestBody{
				Name:          "Valid Role",
				Description:   "Valid description",
				PermissionIDs: nil,
			},
			wantErr: true,
			errMsg:  "at least one permission is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check that whitespace was trimmed
				assert.Equal(t, strings.TrimSpace(tt.request.Name), tt.request.Name)
				assert.Equal(t, strings.TrimSpace(tt.request.Description), tt.request.Description)
			}
		})
	}
}
