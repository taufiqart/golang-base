package storage

import (
	"log/slog"

	"golang-base/config"
	"golang-base/internal/middleware"
	pkgstorage "golang-base/internal/pkg/storage"

	"github.com/gofiber/fiber/v3"
)

type Module struct {
	handler *Handler
}

func New(cfg *config.Config) *Module {
	if cfg == nil {
		cfg = &config.Config{}
	}

	provider, err := pkgstorage.NewProviderFromAppConfig(cfg)
	if err != nil {
		slog.Warn("failed to initialize storage provider, falling back to disabled provider", "error", err)
		provider = &pkgstorage.DisabledProvider{}
	}

	service := NewService(provider)
	handler := NewHandler(service, cfg.MaxUploadSize)

	return &Module{
		handler: handler,
	}
}

func (m *Module) Register(router fiber.Router) {
	storageGroup := router.Group("/storage")

	// Public/System health check for storage
	storageGroup.Get("/ping", m.handler.Ping)

	// Authenticated routes protected by RBAC permissions
	authGroup := storageGroup.Group("", middleware.AuthMiddleware())
	authGroup.Post("/upload", middleware.AllowedPermissions("storage.upload"), m.handler.Upload)
	authGroup.Get("/presigned", middleware.AllowedPermissions("storage.view"), m.handler.GetPresignedURL)
	authGroup.Delete("", middleware.AllowedPermissions("storage.delete"), m.handler.Delete)
}
