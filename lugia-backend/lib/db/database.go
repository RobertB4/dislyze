package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"lugia/lib/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB(env *config.Env) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dsn string

	dsn = fmt.Sprintf(
		"user=%s password=%s host=%s port=5432 dbname=%s sslmode=%s",
		env.DBUser, env.DBPassword, env.DBHost, env.DBName, env.DBSSLMode,
	)

	pgxConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing database config: %w", err)
	}

	pgxConfig.MaxConns = 25
	pgxConfig.MinConns = 5
	pgxConfig.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to database")
	return pool, nil
}

func CloseDB(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
