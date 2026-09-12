package socketio

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Publish(msg Message) bool {
	args := m.Called(msg)
	return args.Bool(0)
}

func (m *mockService) SendToUser(userID string, event string, payload any) bool {
	args := m.Called(userID, event, payload)
	return args.Bool(0)
}

func (m *mockService) Broadcast(event string, payload any) bool {
	args := m.Called(event, payload)
	return args.Bool(0)
}

func (m *mockService) BroadcastToRoom(room string, event string, payload any) bool {
	args := m.Called(room, event, payload)
	return args.Bool(0)
}

func (m *mockService) Channel() chan<- Message {
	return nil
}

func (m *mockService) Start() {}
func (m *mockService) Stop()  {}

func TestHandler_Broadcast_Validation(t *testing.T) {
	svc := &mockService{}
	h := newHandler(svc)

	app := fiber.New()
	app.Post("/broadcast", h.Broadcast)

	t.Run("missing event and payload", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{})
		req := httptest.NewRequest("POST", "/broadcast", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("successful broadcast", func(t *testing.T) {
		svc.On("Publish", mock.Anything).Return(true).Once()

		body, _ := json.Marshal(map[string]any{
			"event":   "test:event",
			"room":    "test:room",
			"payload": "test payload",
		})
		req := httptest.NewRequest("POST", "/broadcast", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
