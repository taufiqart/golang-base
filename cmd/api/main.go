package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"golang-base/config"
	"golang-base/internal/app"
	"golang-base/internal/database"
	"golang-base/internal/pkg/logger"
	"golang-base/internal/pkg/tracing"
)

func main() {
	// 1. Load configuration
	cfg := config.LoadConfig()

	// 2. Initialize the structured logger first so every later boot step is
	// captured in the log files

	ctx := context.Background()

	// 3. Install the trace provider before the database, so the Bun query hook
	// resolves a real tracer on the very first query.
	shutdownTracing, err := tracing.Setup(ctx, tracing.Config{
		Enabled:     cfg.TracingEnabled,
		ServiceName: cfg.AppService,
		Version:     cfg.AppVersion,
		Environment: cfg.AppEnvironment,
		SampleRatio: cfg.TracingSample,
	})
	if err != nil {
		slog.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}

	// 4. Initialize database connections
	db := database.InitPostgres(cfg)
	if db != nil {
		defer database.Close()
	}

	redis := database.InitRedis(cfg)
	if redis != nil {
		defer database.CloseRedis()
	}

	// 5. Create and start Fiber app
	application := app.New(cfg)

	slog.Info("starting server",
		"port", cfg.Port,
		"service", cfg.AppService,
		"version", cfg.AppVersion,
		"log_level", logger.LevelName(logger.Level()),
		"tracing", cfg.TracingEnabled,
	)

	// ponytail: no graceful-shutdown signal handler, so spans still buffered in
	// the batch processor can be lost on SIGKILL. Add a signal.Notify +
	// app.Shutdown step when zero-downtime deploys matter; call shutdownTracing
	// from there instead of only on the error path.
	if err := application.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		slog.Error("failed to start server", "error", err)
		if err := shutdownTracing(context.Background()); err != nil {
			slog.Error("failed to flush traces", "error", err)
		}
		os.Exit(1)
	}

	if err := shutdownTracing(context.Background()); err != nil {
		slog.Error("failed to flush traces", "error", err)
	}
}
