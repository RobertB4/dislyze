package handlers

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"lugia/lib/errors"
	"lugia/queries"

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
					Name:      pgtype.Text{String: "Test User One", Valid: true},
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
					Name:      pgtype.Text{Valid: false},
					Role:      "user",
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
					Role:      "user",
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
					Name:      pgtype.Text{String: "Invalid", Valid: true},
					Role:      "user",
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
					Name:      pgtype.Text{String: "User One", Valid: true},
					Role:      "user",
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
