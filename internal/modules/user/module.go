package user

import (
	"github.com/gofiber/fiber/v2"
)

// Module represents the user module
type Module struct{}

// New creates a new user module instance
func New() *Module {
	return &Module{}
}

// Register registers all user routes to the Fiber app
func (m *Module) Register(router fiber.Router) {
	// User routes are registered in the auth module with proper auth middleware
}
