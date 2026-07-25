package database

import (
	"context"
	"log"

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
		log.Printf("Redis connection failed: %v (Redis will be disabled)", err)
		return nil
	}

	log.Println("Redis connection successfully established")
	Redis = client
	return client
}

// CloseRedis closes the Redis connection
func CloseRedis() {
	if Redis != nil {
		Redis.Close()
	}
}
