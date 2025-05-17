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

// initDB initializes the database connection
func initDB(t *testing.T) *pgxpool.Pool {
	if dbPool != nil {
		return dbPool
	}

	// Get database connection from environment
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

// CleanupDB deletes all data from the database tables using delete.sql
func CleanupDB(t *testing.T) {
	t.Helper()

	pool := initDB(t)

	// Read delete.sql file
	deleteSQL, err := os.ReadFile("/database/delete.sql")
	if err != nil {
		t.Fatalf("Failed to read delete.sql: %v", err)
	}

	// Execute delete.sql
	_, err = pool.Exec(context.Background(), string(deleteSQL))
	if err != nil {
		t.Fatalf("Failed to execute delete.sql: %v", err)
	}
}

// CloseDB closes the database connection
func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
		dbPool = nil
	}
}
