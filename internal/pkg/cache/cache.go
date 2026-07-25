package cache

import (
	"context"
	"encoding/json"
	"time"

	"golang-base/internal/database"

	"github.com/redis/go-redis/v9"
)

// Set stores a value in Redis with the specified TTL (Time-To-Live).
// If Redis is not available, it behaves as a no-op and returns nil.
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if database.Redis == nil {
		return nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return database.Redis.Set(ctx, key, bytes, expiration).Err()
}

// Get retrieves a value from Redis and unmarshals it into the generic type T.
// If the key does not exist or Redis is disabled, it returns redis.Nil error.
func Get[T any](ctx context.Context, key string) (T, error) {
	var result T
	if database.Redis == nil {
		return result, redis.Nil
	}

	val, err := database.Redis.Get(ctx, key).Bytes()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(val, &result)
	return result, err
}

// Delete removes a key from the cache.
func Delete(ctx context.Context, key string) error {
	if database.Redis == nil {
		return nil
	}
	return database.Redis.Del(ctx, key).Err()
}

// Remember fetches a value from cache if it exists.
// If it does not exist, it executes the provided fallback function, stores its result in cache,
// and then returns the result. This is equivalent to Laravel's Cache::remember().
func Remember[T any](ctx context.Context, key string, expiration time.Duration, fallback func() (T, error)) (T, error) {
	// 1. Try to get from cache first
	cached, err := Get[T](ctx, key)
	if err == nil {
		return cached, nil
	}

	// 2. If Redis is disabled, key is missing (redis.Nil), or unmarshal fails, we call the fallback
	fresh, err := fallback()
	if err != nil {
		return fresh, err // Do not cache if fallback returns an error
	}

	// 3. Cache the fresh result
	_ = Set(ctx, key, fresh, expiration)

	return fresh, nil
}
