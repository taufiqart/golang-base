package socketio

import (
	"net/http/httptest"
	"testing"
	"time"

	jwtpkg "golang-base/internal/pkg/jwt"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zishang520/socket.io/v2/socket"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "/socket.io/", cfg.Path)
	assert.Equal(t, 20*time.Second, cfg.PingTimeout)
	assert.Equal(t, 25*time.Second, cfg.PingInterval)
	assert.Equal(t, int64(1024*1024), cfg.MaxPayload)
	assert.Equal(t, "socket.io", cfg.RedisPrefix)
}

func TestNewServer_Default(t *testing.T) {
	srv, err := NewServer(nil)
	require.NoError(t, err)
	require.NotNil(t, srv)
	assert.NotNil(t, srv.Raw())
	assert.NotNil(t, srv.Handler())

	// Test rooms and emitting helper methods
	assert.NotNil(t, srv.To("room-1"))
	assert.NotNil(t, srv.In("room-1"))
	assert.NotNil(t, srv.Of("/admin"))

	// Close cleanly
	err = srv.Close()
	assert.NoError(t, err)
}

func TestNewServer_WithCustomConfig(t *testing.T) {
	cfg := &Config{
		Path:         "/custom-socket.io/",
		PingTimeout:  10 * time.Second,
		PingInterval: 15 * time.Second,
		MaxPayload:   512 * 1024,
		CORS: &CORSConfig{
			Origin:      "*",
			Methods:     []string{"GET", "POST"},
			Headers:     []string{"Authorization"},
			Credentials: true,
		},
		JWTSecret: "test-jwt-secret",
	}

	srv, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, srv)
	defer srv.Close()

	assert.NotNil(t, srv.Raw())
}

func TestNewServer_WithRedisClient(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	cfg := &Config{
		RedisClient: rdb,
		RedisPrefix: "test-socket.io",
	}

	srv, err := NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, srv)
	defer srv.Close()
}

func TestExtractToken(t *testing.T) {
	t.Run("nil handshake", func(t *testing.T) {
		assert.Equal(t, "", ExtractToken(nil))
	})

	t.Run("from auth.token", func(t *testing.T) {
		hs := &socket.Handshake{
			Auth: map[string]any{
				"token": "token-123",
			},
		}
		assert.Equal(t, "token-123", ExtractToken(hs))
	})

	t.Run("from auth.accessToken with Bearer prefix", func(t *testing.T) {
		hs := &socket.Handshake{
			Auth: map[string]any{
				"accessToken": "Bearer token-456",
			},
		}
		assert.Equal(t, "token-456", ExtractToken(hs))
	})

	t.Run("from headers authorization", func(t *testing.T) {
		hs := &socket.Handshake{
			Headers: map[string][]string{
				"authorization": {"Bearer header-token-789"},
			},
		}
		assert.Equal(t, "header-token-789", ExtractToken(hs))
	})

	t.Run("from query token", func(t *testing.T) {
		hs := &socket.Handshake{
			Query: map[string][]string{
				"token": {"query-token-abc"},
			},
		}
		assert.Equal(t, "query-token-abc", ExtractToken(hs))
	})
}

func TestAuthenticateHandshake(t *testing.T) {
	secret := "test-secret-key-for-socketio"

	jwtService := &jwtpkg.JWT{
		Secret:          secret,
		AccessExpireMin: 15,
	}
	token, err := jwtService.GenerateAccessToken("user-uuid-123")
	require.NoError(t, err)

	t.Run("missing token", func(t *testing.T) {
		claims, err := AuthenticateHandshake(nil, secret)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "missing token")
	})

	t.Run("invalid token", func(t *testing.T) {
		hs := &socket.Handshake{
			Auth: map[string]any{
				"token": "invalid.jwt.token",
			},
		}
		claims, err := AuthenticateHandshake(hs, secret)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "invalid or expired token")
	})

	t.Run("valid token", func(t *testing.T) {
		hs := &socket.Handshake{
			Auth: map[string]any{
				"token": token,
			},
		}
		claims, err := AuthenticateHandshake(hs, secret)
		assert.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, "user-uuid-123", claims.UserID)
	})
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	middleware := JWTMiddleware("test-secret")
	client := socket.MakeSocket()

	var middlewareErr *socket.ExtendedError
	middleware(client, func(err *socket.ExtendedError) {
		middlewareErr = err
	})

	assert.NotNil(t, middlewareErr)
	assert.Contains(t, middlewareErr.Error(), "missing token")
}

func TestGetClaimsAndGetUserID(t *testing.T) {
	client := socket.MakeSocket()

	// Initially empty
	claims, ok := GetClaims(client)
	assert.False(t, ok)
	assert.Nil(t, claims)

	userID, ok := GetUserID(client)
	assert.False(t, ok)
	assert.Equal(t, "", userID)

	// Set claims
	expectedClaims := &jwtpkg.JWTClaims{
		UserID: "user-456",
	}
	client.SetData(expectedClaims)

	claims, ok = GetClaims(client)
	assert.True(t, ok)
	assert.Equal(t, "user-456", claims.UserID)

	userID, ok = GetUserID(client)
	assert.True(t, ok)
	assert.Equal(t, "user-456", userID)
}

func TestServer_RegisterFiber(t *testing.T) {
	srv, err := NewServer(nil)
	require.NoError(t, err)
	defer srv.Close()

	app := fiber.New()
	srv.Register(app)

	// Socket.IO polling handshake request
	req := httptest.NewRequest("GET", "/socket.io/?EIO=4&transport=polling", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5000})
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)
}
