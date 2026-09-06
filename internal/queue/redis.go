package queue

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Config holds the settings needed to connect to Redis. Addr is either a
// plain "host:port" (local dev) or a full connection URL such as
// "rediss://user:pass@host:port" (managed hosts like Upstash).
type Config struct {
	Addr string
}

// DefaultConfig returns connection settings matching the docker-compose.yml
// defaults used for local development.
func DefaultConfig() Config {
	return Config{Addr: "localhost:6379"}
}

// AddrFromEnv returns the Redis address to use: REDIS_URL if set — the form
// managed hosts like Upstash hand you (e.g. "rediss://default:pass@host:port")
// — otherwise the local dev DefaultConfig().
func AddrFromEnv() string {
	if addr := os.Getenv("REDIS_URL"); addr != "" {
		return addr
	}
	return DefaultConfig().Addr
}

// NewClient creates and verifies a Redis client connection.
func NewClient(ctx context.Context, cfg Config) (*redis.Client, error) {
	opts, err := parseAddr(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("invalid redis address: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func parseAddr(addr string) (*redis.Options, error) {
	if strings.Contains(addr, "://") {
		return redis.ParseURL(addr)
	}
	return &redis.Options{Addr: addr}, nil
}
