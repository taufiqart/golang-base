package socketio

import (
	"golang-base/internal/middleware"
	"golang-base/internal/pkg/response"
	"golang-base/internal/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

type handler struct {
	service Service
}

func newHandler(service Service) *handler {
	return &handler{
		service: service,
	}
}

// RegisterRoutes registers the administrative HTTP routes for Socket.IO.
func (h *handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/socketio", middleware.AuthMiddleware())
	group.Post("/broadcast", h.Broadcast)
}

// Broadcast handles POST /api/v1/socketio/broadcast
func (h *handler) Broadcast(c fiber.Ctx) error {
	var req BroadcastRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid request payload")
	}

	if errs := validator.ValidateStruct(&req); len(errs) > 0 {
		return response.BadRequestValidation(c, errs)
	}

	ok := h.service.Publish(Message{
		Namespace: req.Namespace,
		Room:      req.Room,
		Event:     req.Event,
		Payload:   req.Payload,
	})

	if !ok {
		return response.InternalError(c, "Failed to queue message: channel buffer is full")
	}

	return response.OK(c, fiber.Map{
		"message":   "Message broadcast queued",
		"namespace": req.Namespace,
		"event":     req.Event,
		"room":      req.Room,
	})
}
