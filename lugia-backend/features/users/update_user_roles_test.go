package users

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestUpdateUserRolesRequestBody_Validate(t *testing.T) {
	// Helper function to create a UUID from string
	mustParseUUID := func(s string) pgtype.UUID {
		var uuid pgtype.UUID
		err := uuid.Scan(s)
		if err != nil {
			t.Fatalf("Failed to parse UUID %s: %v", s, err)
		}
		return uuid
	}

	tests := []struct {
		name    string
		request UpdateUserRolesRequestBody
		wantErr bool
		errMsg  string
	}{
		{
			name: "single valid role ID",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []pgtype.UUID{
					mustParseUUID("e0000000-0000-0000-0000-000000000001"),
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid role IDs",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []pgtype.UUID{
					mustParseUUID("e0000000-0000-0000-0000-000000000001"),
					mustParseUUID("e0000000-0000-0000-0000-000000000002"),
				},
			},
			wantErr: false,
		},
		{
			name: "empty role IDs array",
			request: UpdateUserRolesRequestBody{
				RoleIDs: []pgtype.UUID{},
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
