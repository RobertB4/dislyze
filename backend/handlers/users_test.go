package handlers

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"lugia/lib/errors"
	"lugia/queries"

	"github.com/stretchr/testify/assert"

	"github.com/jackc/pgx/v5/pgtype"
)

func newPgtypeUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	err := u.Scan(s)
	if err != nil {
		panic("Failed to scan UUID string in test helper: " + err.Error())
	}
	return u
}

func TestMapDBUsersToResponse(t *testing.T) {
	// Use a fixed time for CreatedAt/UpdatedAt consistency in tests,
	// truncating to avoid nanosecond precision issues in comparisons.
	now := time.Now().Truncate(time.Second)

	uuid1Str := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	uuid2Str := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
	uuidInvalidBytes := [16]byte{} // Zero UUID for the invalid ID case

	pUUID1 := newPgtypeUUID(uuid1Str)
	pUUID2 := newPgtypeUUID(uuid2Str)

	tests := []struct {
		name      string
		input     []*queries.GetUsersByTenantIDRow
		wantUsers []User
		wantErr   error
	}{
		{
			name:      "empty input",
			input:     []*queries.GetUsersByTenantIDRow{},
			wantUsers: []User{},
			wantErr:   nil,
		},
		{
			name: "single user all fields valid",
			input: []*queries.GetUsersByTenantIDRow{
				{
					ID:        pUUID1,
					Email:     "test1@example.com",
					Name:      "Test User One",
					Role:      "admin",
					Status:    "active",
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
			},
			wantUsers: []User{
				{
					ID:        uuid1Str,
					Email:     "test1@example.com",
					Name:      "Test User One",
					Role:      "admin",
					Status:    "active",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			wantErr: nil,
		},
		{
			name: "single user with null name, status active",
			input: []*queries.GetUsersByTenantIDRow{
				{
					ID:        pUUID2,
					Email:     "test2@example.com",
					Name:      "",
					Role:      "editor",
					Status:    "active",
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
			},
			wantUsers: []User{
				{
					ID:        uuid2Str,
					Email:     "test2@example.com",
					Name:      "",
					Role:      "editor",
					Status:    "active",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			wantErr: nil,
		},
		{
			name: "user with invalid database ID",
			input: []*queries.GetUsersByTenantIDRow{
				{
					ID:        pgtype.UUID{Bytes: uuidInvalidBytes, Valid: false},
					Email:     "invalidid@example.com",
					Name:      "Invalid",
					Role:      "editor",
					Status:    "active",
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
			},
			wantUsers: nil,
			wantErr:   fmt.Errorf("%w: user record with invalid/NULL ID (email for context: invalidid@example.com)", ErrInvalidUserDataFromDB),
		},
		{
			name: "input slice with a nil pointer element",
			input: []*queries.GetUsersByTenantIDRow{
				{
					ID:        pUUID1,
					Email:     "user1@example.com",
					Name:      "User One",
					Role:      "editor",
					Status:    "active",
					CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
				nil,
			},
			wantUsers: nil,
			wantErr:   fmt.Errorf("%w: encountered nil user record at index %d", ErrInvalidUserDataFromDB, 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsers, gotErr := mapDBUsersToResponse(tt.input)

			if tt.wantErr != nil {
				if gotErr == nil {
					t.Fatalf("mapDBUsersToResponse() error = nil, wantErr %v", tt.wantErr)
				}
				if !errors.Is(gotErr, ErrInvalidUserDataFromDB) {
					t.Errorf("mapDBUsersToResponse() gotErr (%v) does not wrap expected ErrInvalidUserDataFromDB", gotErr)
				}
				if gotErr.Error() != tt.wantErr.Error() {
					t.Errorf("mapDBUsersToResponse() error message = %q, wantErrMsg %q", gotErr.Error(), tt.wantErr.Error())
				}
			} else if gotErr != nil {
				t.Fatalf("mapDBUsersToResponse() unexpected error = %v", gotErr)
			}

			if !reflect.DeepEqual(gotUsers, tt.wantUsers) {
				t.Errorf("mapDBUsersToResponse() gotUsers = %v, want %v", gotUsers, tt.wantUsers)
			}
		})
	}
}

func TestInviteUserRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request InviteUserRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  "editor",
			},
			wantErr: nil,
		},
		{
			name: "empty email",
			request: InviteUserRequest{
				Email: "",
				Name:  "Test User",
				Role:  "editor",
			},
			wantErr: fmt.Errorf("email is required"),
		},
		{
			name: "invalid email format (basic check)",
			request: InviteUserRequest{
				Email: "testexample.com",
				Name:  "Test User",
				Role:  "editor",
			},
			wantErr: fmt.Errorf("email is invalid"),
		},
		{
			name: "empty name",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "",
				Role:  "editor",
			},
			wantErr: fmt.Errorf("name is required and cannot be only whitespace"),
		},
		{
			name: "name with only whitespace",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "   ",
				Role:  "editor",
			},
			wantErr: fmt.Errorf("name is required and cannot be only whitespace"),
		},
		{
			name: "empty role",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  "",
			},
			wantErr: fmt.Errorf("role is required"),
		},
		{
			name: "invalid role value",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  "guest",
			},
			wantErr: fmt.Errorf("role is invalid, must be 'admin' or 'editor'"),
		},
		{
			name: "role with mixed case (should be normalized and valid)",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User",
				Role:  "Admin",
			},
			wantErr: nil, // Expecting it to be normalized to lowercase "editor"
		},
		{
			name: "role with leading/trailing whitespace (should be normalized and valid)",
			request: InviteUserRequest{
				Email: "test@example.com",
				Name:  "Test User With Spaces   ",
				Role:  "  editor  ",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCopy := tt.request // Copy to avoid modifying the original tt.request due to TrimSpace
			gotErr := reqCopy.Validate()

			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("%s: Validate() error = nil, wantErr %v", tt.name, tt.wantErr)
					return
				}
				if gotErr.Error() != tt.wantErr.Error() {
					t.Errorf("%s: Validate() error message = %q, wantErrMsg %q", tt.name, gotErr.Error(), tt.wantErr.Error())
				}
			} else if gotErr != nil {
				t.Errorf("%s: Validate() unexpected error = %v", tt.name, gotErr)
			}

			// Additionally, check if fields were trimmed as expected, for cases where no error is expected
			if tt.wantErr == nil && tt.name == "role with leading/trailing whitespace (should be normalized and valid)" {
				expectedTrimmedName := "Test User With Spaces"
				expectedTrimmedRole := "editor"
				if reqCopy.Name != expectedTrimmedName {
					t.Errorf("%s: Name not trimmed as expected: got %q, want %q", tt.name, reqCopy.Name, expectedTrimmedName)
				}
				if reqCopy.Role != expectedTrimmedRole {
					t.Errorf("%s: Role not trimmed/lowercased as expected: got %q, want %q", tt.name, reqCopy.Role, expectedTrimmedRole)
				}
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
