package app

import (
	"golang-base/config"
	"golang-base/internal/database"
	"golang-base/internal/middleware"
	"golang-base/internal/modules/auth"
	"golang-base/internal/modules/docs"
	"golang-base/internal/modules/storage"
	"golang-base/internal/modules/user"
	"golang-base/internal/pkg/logger"
	pkgstorage "golang-base/internal/pkg/storage"

	"github.com/gofiber/fiber/v3"
)

func New(cfg *config.Config) *fiber.App {
	if cfg == nil { cfg = &config.Config{AppService: "golang-base", AppVersion: "dev", LogLevel: "info"} }
	logger.Setup(cfg.LogLevel, cfg.LogPretty)
	serviceName := cfg.AppService
	if serviceName == "" { serviceName = "golang-base" }
	bodyLimit := 10 * 1024 * 1024
	if cfg.MaxUploadSize > 0 { bodyLimit = int(cfg.MaxUploadSize) }
	app := fiber.New(fiber.Config{
		AppName: serviceName, ErrorHandler: customErrorHandler,
		ReadBufferSize: 32 * 1024, BodyLimit: bodyLimit,
	})
	middleware.SetupMiddleware(app, cfg)
	storageProvider, _ := pkgstorage.NewProviderFromAppConfig(cfg)
	app.Get("/health", func(c fiber.Ctx) error {
		status := fiber.Map{"status": "healthy", "service": serviceName}
		if database.DB != nil { if err := database.DB.Ping(); err != nil { status["database"] = "disconnected" } else { status["database"] = "connected" } } else { status["database"] = "disabled" }
		if database.Redis != nil { if err := database.Redis.Ping(c.Context()).Err(); err != nil { status["redis"] = "disconnected" } else { status["redis"] = "connected" } } else { status["redis"] = "disabled" }
		if storageProvider != nil { if err := storageProvider.Ping(c.Context()); err != nil { status["storage"] = "disconnected" } else { status["storage"] = "connected" } } else { status["storage"] = "disabled" }
		return c.JSON(status)
	})
	apiGroup := app.Group("/api/v1")
	docs.New().Register(apiGroup)
	auth.New().Register(apiGroup)
	user.New().Register(apiGroup)
	storage.New(cfg).Register(apiGroup)
	return app
}

func customErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"
	if e, ok := err.(*fiber.Error); ok { code = e.Code; if e.Message != "" { message = e.Message } }
	switch code {
	case fiber.StatusBadRequest: message = "Bad request"
	case fiber.StatusUnauthorized: message = "Unauthorized"
	case fiber.StatusForbidden: message = "Forbidden"
	case fiber.StatusNotFound: message = "Resource not found"
	case fiber.StatusConflict: message = "Resource conflict"
	case fiber.StatusUnprocessableEntity: message = "Unprocessable entity"
	case fiber.StatusRequestEntityTooLarge: message = "Request entity too large"
	case fiber.StatusTooManyRequests: message = "Too many requests"
	}
	return c.Status(code).JSON(fiber.Map{"code": code, "message": message})
}
