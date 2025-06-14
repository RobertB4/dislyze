package db

import (
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func RunMigrations(pool *pgxpool.Pool) error {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			delay := time.Duration(attempt-1) * baseDelay
			log.Printf("Migration attempt %d failed, retrying in %v...", attempt-1, delay)
			time.Sleep(delay)
		}

		err := attemptMigration(pool)
		if err == nil {
			log.Println("Successfully ran database migrations")
			return nil
		}

		lastErr = err
		log.Printf("Migration attempt %d failed: %v", attempt, err)
	}

	return fmt.Errorf("failed to run migrations after %d attempts: %w", maxRetries, lastErr)
}

func attemptMigration(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database connection: %v", err)
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, "../database/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
