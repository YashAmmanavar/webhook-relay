package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds the settings needed to connect to Postgres.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultConfig returns connection settings matching the docker-compose.yml
// defaults used for local development.
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     "5433",
		User:     "webhookrelay",
		Password: "devpassword",
		DBName:   "webhookrelay",
		SSLMode:  "disable",
	}
}

// connString builds a libpq-style connection string from the config.
func (c Config) connString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// ConnStringFromEnv returns the Postgres connection string to use:
// DATABASE_URL if set — the form managed hosts like Neon or Supabase hand
// you (e.g. "postgresql://user:pass@host/db?sslmode=require") — otherwise
// the local dev DefaultConfig().
func ConnStringFromEnv() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return DefaultConfig().connString()
}

// NewPool creates and verifies a Postgres connection pool using pgx.
// Call Close() on the returned pool when your application shuts down.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify the connection actually works, not just that the pool was created.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
