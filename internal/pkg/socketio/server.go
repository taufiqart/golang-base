package socketio

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/zishang520/engine.io/v2/types"
	"github.com/zishang520/socket.io-go-redis/adapter"
	redistypes "github.com/zishang520/socket.io-go-redis/types"
	"github.com/zishang520/socket.io/v2/socket"
)

var (
	defaultServerMu sync.RWMutex
	defaultServer   *Server
)

// SetDefault sets the package-level default Server.
func SetDefault(s *Server) {
	defaultServerMu.Lock()
	defer defaultServerMu.Unlock()
	defaultServer = s
}

// GetServer returns the default package-level Server instance.
func GetServer() *Server {
	defaultServerMu.RLock()
	defer defaultServerMu.RUnlock()
	return defaultServer
}

// BroadcastEmit sends an event to all connected clients using the default server.
func BroadcastEmit(event string, data ...any) {
	if s := GetServer(); s != nil {
		s.Emit(event, data...)
	}
}

// BroadcastTo sends an event to a room using the default server.
func BroadcastTo(room string, event string, data ...any) {
	if s := GetServer(); s != nil {
		s.To(room).Emit(event, data...)
	}
}

// isNil checks if an interface is nil, handling typed nil pointers.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

// Server wraps the official Socket.IO v4 server implementation.
type Server struct {
	raw  *socket.Server
	opts *socket.ServerOptions
	path string
}

// NewServer initializes a new Socket.IO Server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := socket.DefaultServerOptions()

	// Path
	path := cfg.Path
	if path == "" {
		path = "/socket.io/"
	}
	opts.SetPath(path)

	// Ping timeout and interval
	if cfg.PingTimeout > 0 {
		opts.SetPingTimeout(cfg.PingTimeout)
	}
	if cfg.PingInterval > 0 {
		opts.SetPingInterval(cfg.PingInterval)
	}

	// Payload limit
	if cfg.MaxPayload > 0 {
		opts.SetMaxHttpBufferSize(cfg.MaxPayload)
	}

	// CORS configuration
	if cfg.CORS != nil {
		cors := &types.Cors{
			Origin:      cfg.CORS.Origin,
			Methods:     cfg.CORS.Methods,
			Headers:     cfg.CORS.Headers,
			Credentials: cfg.CORS.Credentials,
		}
		opts.SetCors(cors)
	}

	// Redis Adapter for horizontal scaling (if provided and non-nil)
	if !isNil(cfg.RedisClient) {
		redisOpts := adapter.DefaultRedisAdapterOptions()
		prefix := cfg.RedisPrefix
		if prefix == "" {
			prefix = "socket.io"
		}
		redisOpts.SetKey(prefix)

		rClient := redistypes.NewRedisClient(context.Background(), cfg.RedisClient)
		opts.SetAdapter(&adapter.RedisAdapterBuilder{
			Redis: rClient,
			Opts:  redisOpts,
		})
	}

	rawServer := socket.NewServer(nil, opts)

	// If JWT secret is specified, automatically register the authentication middleware on root namespace
	if cfg.JWTSecret != "" {
		rawServer.Use(JWTMiddleware(cfg.JWTSecret))
	}

	return &Server{
		raw:  rawServer,
		opts: opts,
		path: path,
	}, nil
}

// Raw returns the underlying *socket.Server instance.
func (s *Server) Raw() *socket.Server {
	return s.raw
}

// OnConnection registers a listener for incoming client connections on the root namespace.
func (s *Server) OnConnection(handler func(client *socket.Socket)) {
	s.raw.On("connection", func(args ...any) {
		if len(args) > 0 {
			if client, ok := args[0].(*socket.Socket); ok {
				handler(client)
			}
		}
	})
}

// Emit broadcasts an event to all connected clients on the root namespace.
func (s *Server) Emit(event string, data ...any) {
	s.raw.Emit(event, data...)
}

// To targets a specific room for subsequent broadcast.
func (s *Server) To(room string) *socket.BroadcastOperator {
	return s.raw.To(socket.Room(room))
}

// In is an alias for To.
func (s *Server) In(room string) *socket.BroadcastOperator {
	return s.raw.In(socket.Room(room))
}

// Of returns a specific namespace.
func (s *Server) Of(namespace string) socket.Namespace {
	return s.raw.Of(namespace, nil)
}

// Use adds a namespace middleware to the root namespace.
func (s *Server) Use(fn socket.NamespaceMiddleware) {
	s.raw.Use(fn)
}

// Handler returns a Fiber-compatible HTTP handler.
func (s *Server) Handler() fiber.Handler {
	return adaptor.HTTPHandler(s.raw.ServeHandler(s.opts))
}

// RegisterRoute mounts the Socket.IO handler on any fiber.Router.
func (s *Server) RegisterRoute(router fiber.Router) {
	handler := s.Handler()
	base := strings.TrimRight(s.path, "/")
	if base == "" {
		base = "/socket.io"
	}
	router.All(base+"/*", handler)
	router.All(base, handler)
}

// Register mounts the Socket.IO handler on the given Fiber app.
func (s *Server) Register(app *fiber.App) {
	s.RegisterRoute(app)
}

// Close gracefully closes the Socket.IO server.
func (s *Server) Close() error {
	done := make(chan error, 1)
	s.raw.Close(func(err error) {
		done <- err
	})

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("socket.io server close timeout")
	}
}
