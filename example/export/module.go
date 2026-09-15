package export

import (
	"github.com/gofiber/fiber/v3"
)

type Module struct {
	handler *Handler
}

func New() *Module {
	handler := NewHandler()
	return &Module{handler: handler}
}

func (m *Module) Register(router fiber.Router) {
	api := router.Group("/export")
	api.Get("/xlsx", m.handler.DownloadXLSX)
	api.Get("/pdf", m.handler.DownloadPDF)
}
