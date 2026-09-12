package socketio

import (
	"log/slog"
	"sync"

	pkgsocketio "golang-base/internal/pkg/socketio"

	"github.com/zishang520/socket.io/v2/socket"
)

// Service defines the contract for sending real-time messages via Go channels to Socket.IO clients.
type Service interface {
	// Publish pushes a Message onto the internal channel (non-blocking). Returns false if buffer is full.
	Publish(msg Message) bool

	// SendToUser pushes an event directly targeted to a user's private room ("user:<userID>").
	SendToUser(userID string, event string, payload any) bool

	// Broadcast pushes an event to all connected Socket.IO clients.
	Broadcast(event string, payload any) bool

	// BroadcastToRoom pushes an event to all clients in a specific room.
	BroadcastToRoom(room string, event string, payload any) bool

	// Channel returns the send-only channel for publishing messages directly.
	Channel() chan<- Message

	// Start spawns the background worker goroutine to process messages from the channel.
	Start()

	// Stop gracefully shuts down the background worker.
	Stop()
}

type service struct {
	server   *pkgsocketio.Server
	msgChan  chan Message
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
	mu       sync.Mutex
}

// NewService creates a new Service instance with a buffered channel.
func NewService(server *pkgsocketio.Server, bufferSize int) Service {
	if bufferSize <= 0 {
		bufferSize = 1024
	}

	return &service{
		server:   server,
		msgChan:  make(chan Message, bufferSize),
		stopChan: make(chan struct{}),
	}
}

func (s *service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}
	s.started = true

	s.wg.Add(1)
	go s.worker()
}

func (s *service) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *service) Channel() chan<- Message {
	return s.msgChan
}

func (s *service) Publish(msg Message) bool {
	select {
	case s.msgChan <- msg:
		return true
	default:
		slog.Warn("Socket.IO channel buffer full, dropped real-time message",
			"event", msg.Event,
			"room", msg.Room,
		)
		return false
	}
}

func (s *service) SendToUser(userID string, event string, payload any) bool {
	return s.Publish(Message{
		Room:    "user:" + userID,
		Event:   event,
		Payload: payload,
	})
}

func (s *service) Broadcast(event string, payload any) bool {
	return s.Publish(Message{
		Event:   event,
		Payload: payload,
	})
}

func (s *service) BroadcastToRoom(room string, event string, payload any) bool {
	return s.Publish(Message{
		Room:    room,
		Event:   event,
		Payload: payload,
	})
}

// worker drains messages from msgChan and emits them through the Socket.IO server.
func (s *service) worker() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			// Drain remaining messages before exiting
			for {
				select {
				case msg := <-s.msgChan:
					s.dispatch(msg)
				default:
					return
				}
			}
		case msg := <-s.msgChan:
			s.dispatch(msg)
		}
	}
}

func (s *service) dispatch(msg Message) {
	if s.server == nil {
		return
	}

	if msg.Namespace != "" && msg.Namespace != "/" {
		nsp := s.server.Of(msg.Namespace)
		if msg.Room != "" {
			nsp.To(socket.Room(msg.Room)).Emit(msg.Event, msg.Payload)
		} else {
			nsp.Emit(msg.Event, msg.Payload)
		}
		return
	}

	if msg.Room != "" {
		s.server.To(msg.Room).Emit(msg.Event, msg.Payload)
	} else {
		s.server.Emit(msg.Event, msg.Payload)
	}
}
