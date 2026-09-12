package socketio

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// CORSConfig holds CORS options for the Socket.IO server.
type CORSConfig struct {
	Origin      any      // e.g. "*", []string{"http://localhost:3000"}
	Methods     []string // e.g. []string{"GET", "POST"}
	Headers     []string // e.g. []string{"Authorization", "Content-Type"}
	Credentials bool
}

// Config defines the configuration options for the Socket.IO server.
type Config struct {
	// Path is the URL prefix for Socket.IO requests (default: "/socket.io/")
	Path string

	// CORS configuration for incoming client connections
	CORS *CORSConfig

	// PingTimeout is the duration after which a client without pong is considered dead (default: 20s)
	PingTimeout time.Duration

	// PingInterval is how often ping packets are sent to connected clients (default: 25s)
	PingInterval time.Duration

	// MaxPayload is the maximum allowed message payload size in bytes (default: 1MB)
	MaxPayload int64

	// RedisClient enables the Redis Adapter for horizontal multi-server clustering when provided
	RedisClient redis.UniversalClient

	// RedisPrefix is the channel prefix used by the Redis Adapter (default: "socket.io")
	RedisPrefix string

	// JWTSecret is the secret key used for JWT handshake verification (optional)
	JWTSecret string
}

// DefaultConfig returns a Config instance with standard production defaults.
func DefaultConfig() *Config {
	return &Config{
		Path:         "/socket.io/",
		PingTimeout:  20 * time.Second,
		PingInterval: 25 * time.Second,
		MaxPayload:   1024 * 1024, // 1MB
		RedisPrefix:  "socket.io",
	}
}
