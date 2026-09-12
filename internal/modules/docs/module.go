package docs

import (
	"github.com/gofiber/fiber/v3"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Register(router fiber.Router) {
	handler := newHandler()
	handler.RegisterRoutes(router)
}
