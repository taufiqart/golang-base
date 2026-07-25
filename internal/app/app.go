package app

import (
	"log/slog"

	"golang-base/config"
	"golang-base/internal/database"
	"golang-base/internal/middleware"
	"golang-base/internal/modules/auth"
	"golang-base/internal/modules/docs"
	"golang-base/internal/modules/user"
	"golang-base/internal/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

// New creates and configures the Fiber application with all routes.
// Database connections must be initialized before calling this function.
func New(cfg *config.Config) *fiber.App {
	// Setup structured JSON logger
	logger.Setup()

	// Setup Fiber app with custom error handler
	app := fiber.New(fiber.Config{
		AppName:        "golang-base",
		ErrorHandler:   customErrorHandler,
		ReadBufferSize: 32 * 1024,
	})

	// Setup global middleware
	middleware.SetupMiddleware(app, cfg)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		status := fiber.Map{
			"status":  "healthy",
			"service": "golang-base",
		}

		if database.DB != nil {
			if err := database.DB.Ping(); err != nil {
				status["database"] = "disconnected"
			} else {
				status["database"] = "connected"
			}
		} else {
			status["database"] = "disabled"
		}

		if database.Redis != nil {
			if err := database.Redis.Ping(c.Context()).Err(); err != nil {
				status["redis"] = "disconnected"
			} else {
				status["redis"] = "connected"
			}
		} else {
			status["redis"] = "disabled"
		}

		return c.JSON(status)
	})

	// Register modules
	apiGroup := app.Group("/api/v1")
	docs.New().Register(apiGroup)
	auth.New().Register(apiGroup)
	user.New().Register(apiGroup)

	return app
}

// customErrorHandler provides consistent error responses per OpenAPI spec
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		if e.Message != "" {
			message = e.Message
		}
	}

	if message == "Internal server error" {
		switch code {
		case fiber.StatusBadRequest:
			message = "Bad request"
		case fiber.StatusUnauthorized:
			message = "Unauthorized"
		case fiber.StatusForbidden:
			message = "Forbidden"
		case fiber.StatusNotFound:
			message = "Not found"
		case fiber.StatusMethodNotAllowed:
			message = "Method not allowed"
		case fiber.StatusRequestTimeout:
			message = "Request timeout"
		case fiber.StatusConflict:
			message = "Conflict"
		}
	}

	// Log the original error if it's a 5xx error
	if code >= fiber.StatusInternalServerError {
		slog.Error("Internal server error", "method", c.Method(), "path", c.Path(), "error", err.Error())
	}

	return c.Status(code).JSON(fiber.Map{
		"message": message,
	})
}
