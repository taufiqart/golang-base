package database

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"golang-base/config"
)

// Redis holds the Redis client instance
var Redis *redis.Client

// InitRedis initializes Redis connection
func InitRedis(cfg *config.Config) *redis.Client {
	addr := cfg.RedisAddr
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("redis connection failed, redis will be disabled", "error", err)
		return nil
	}

	slog.Info("redis connection established")
	Redis = client
	return client
}

// RedisPoolStats reports connection pool statistics, or nil when Redis is
// disabled. It exists so callers such as the metrics registry can read the pool
// without importing go-redis themselves.
func RedisPoolStats() *redis.PoolStats {
	if Redis == nil {
		return nil
	}
	return Redis.PoolStats()
}

// CloseRedis closes the Redis connection
func CloseRedis() {
	if Redis != nil {
		Redis.Close()
	}
}
