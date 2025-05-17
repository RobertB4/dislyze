package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
)

func initDB(t *testing.T) *pgxpool.Pool {
	if dbPool != nil {
		return dbPool
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"test_user",
		"test_password",
		"postgres",
		"5432",
		"test_db",
	)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	dbPool = pool
	return pool
}

func CleanupDB(t *testing.T) {
	t.Helper()

	pool := initDB(t)

	deleteSQL, err := os.ReadFile("/database/delete.sql")
	if err != nil {
		t.Fatalf("Failed to read delete.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(deleteSQL))
	if err != nil {
		t.Fatalf("Failed to execute delete.sql: %v", err)
	}
}

func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
		dbPool = nil
	}
}
