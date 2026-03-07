package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateUserRolesRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateUserRolesRequestBody
		wantErr bool
		errMsg  string
	}{
		{
			name: "single valid role ID",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []string{"e0000000-0000-0000-0000-000000000001"},
			},
			wantErr: false,
		},
		{
			name: "multiple valid role IDs",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []string{
					"e0000000-0000-0000-0000-000000000001",
					"e0000000-0000-0000-0000-000000000002",
				},
			},
			wantErr: false,
		},
		{
			name: "empty role IDs array",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []string{},
			},
			wantErr: true,
			errMsg:  "users need at least one role",
		},
		{
			name: "nil role IDs array",
			request: UpdateUserRolesRequestBody{
				RoleIDs: nil,
			},
			wantErr: true,
			errMsg:  "users need at least one role",
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
				// For successful validations, check that we have at least one role
				assert.NotEmpty(t, tt.request.RoleIDs, "Valid request should have at least one role ID")
			}
		})
	}
}
