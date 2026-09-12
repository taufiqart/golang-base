package socketio

import (
	"sync"

	"golang-base/config"
	pkgsocketio "golang-base/internal/pkg/socketio"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/zishang520/socket.io/v2/socket"
)

var (
	moduleMu       sync.RWMutex
	defaultService Service
	defaultServer  *pkgsocketio.Server
)

// GetService returns the global module Service instance.
func GetService() Service {
	moduleMu.RLock()
	defer moduleMu.RUnlock()
	return defaultService
}

// GetServer returns the underlying pkgsocketio.Server instance.
func GetServer() *pkgsocketio.Server {
	moduleMu.RLock()
	defer moduleMu.RUnlock()
	return defaultServer
}

// Channel returns the send-only Go channel from the default Service.
// Other modules can push messages directly to this channel without importing Socket.IO internals.
func Channel() chan<- Message {
	if s := GetService(); s != nil {
		return s.Channel()
	}
	return nil
}

// Publish sends a message to the internal channel via the default Service.
func Publish(msg Message) bool {
	if s := GetService(); s != nil {
		return s.Publish(msg)
	}
	return false
}

// SendToUser pushes an event directly to a user's private room ("user:<userID>").
func SendToUser(userID string, event string, payload any) bool {
	if s := GetService(); s != nil {
		return s.SendToUser(userID, event, payload)
	}
	return false
}

// Broadcast sends an event to all connected clients.
func Broadcast(event string, payload any) bool {
	if s := GetService(); s != nil {
		return s.Broadcast(event, payload)
	}
	return false
}

// BroadcastToRoom sends an event to a specific room.
func BroadcastToRoom(room string, event string, payload any) bool {
	if s := GetService(); s != nil {
		return s.BroadcastToRoom(room, event, payload)
	}
	return false
}

// Of returns or initializes a specific Socket.IO namespace (e.g. "/chat") on the default server.
func Of(namespace string) socket.Namespace {
	if s := GetServer(); s != nil {
		return s.Of(namespace)
	}
	return nil
}

// Module encapsulates the Socket.IO server and asynchronous Channel worker.
type Module struct {
	server  *pkgsocketio.Server
	service Service
	handler *handler
}

// New initializes the Socket.IO Module, setting up the server, channel worker, and default room listeners.
func New(cfg *config.Config, redisClient redis.UniversalClient) (*Module, error) {
	var jwtSecret string
	if cfg != nil {
		jwtSecret = cfg.JWTSecret
	}

	ioServer, err := pkgsocketio.NewServer(&pkgsocketio.Config{
		JWTSecret:   jwtSecret,
		RedisClient: redisClient,
	})
	if err != nil {
		return nil, err
	}

	// Auto join connected users to their own private room
	ioServer.OnConnection(func(client *socket.Socket) {
		if userID, ok := pkgsocketio.GetUserID(client); ok {
			client.Join(socket.Room("user:" + userID))
		}
	})

	svc := NewService(ioServer, 1024)
	svc.Start()

	// Register as global default service and server
	moduleMu.Lock()
	defaultService = svc
	defaultServer = ioServer
	moduleMu.Unlock()

	return &Module{
		server:  ioServer,
		service: svc,
		handler: newHandler(svc),
	}, nil
}

// Service returns the underlying Service instance.
func (m *Module) Service() Service {
	return m.service
}

// Server returns the underlying pkgsocketio.Server.
func (m *Module) Server() *pkgsocketio.Server {
	return m.server
}

// Register registers HTTP REST endpoints under the provided router (typically /api/v1).
func (m *Module) Register(router fiber.Router) {
	m.handler.RegisterRoutes(router)
}

// RegisterSocket mounts the Socket.IO engine endpoints (/socket.io/*) on the root Fiber app.
func (m *Module) RegisterSocket(app *fiber.App) {
	if m.server != nil {
		m.server.Register(app)
	}
}

// Close gracefully terminates the channel worker and the Socket.IO server.
func (m *Module) Close() error {
	if m.service != nil {
		m.service.Stop()
	}
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}
