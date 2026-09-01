package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Config holds the settings needed to connect to Redis.
type Config struct {
	Addr string
}

// DefaultConfig returns connection settings matching the docker-compose.yml
// defaults used for local development.
func DefaultConfig() Config {
	return Config{Addr: "localhost:6379"}
}

// NewClient creates and verifies a Redis client connection.
func NewClient(ctx context.Context, cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: cfg.Addr})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
