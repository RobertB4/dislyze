package utils

import (
	"crypto/rand"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func NewUUID() (pgtype.UUID, error) {
	var pguid pgtype.UUID
	_, err := rand.Read(pguid.Bytes[:])
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to generate random bytes for UUID: %w", err)
	}

	pguid.Valid = true
	return pguid, nil
}