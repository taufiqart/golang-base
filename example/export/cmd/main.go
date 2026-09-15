// Command export runs a minimal server that exposes the export example.
//
// It reads PORT from the environment so integration tests can bind a dynamic
// port, and binds all interfaces so it also works from a container.
package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v3"

	exportexample "golang-base/example/export"
	"golang-base/internal/pkg/logger"
)

func main() {
	logger.Setup(envOr("LOG_LEVEL", "info"))

	app := fiber.New(fiber.Config{AppName: "export-example"})
	exportexample.New().Register(app)

	port := envOr("PORT", "8080")
	slog.Info("starting export example", "port", port)
	if err := app.Listen(":" + port); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
