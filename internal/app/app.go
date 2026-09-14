package app

import (
	"log/slog"

	"golang-base/config"
	"golang-base/internal/database"
	"golang-base/internal/middleware"
	"golang-base/internal/modules/auth"
	"golang-base/internal/modules/docs"
	"golang-base/internal/modules/storage"
	"golang-base/internal/modules/user"
	"golang-base/internal/pkg/logger"
	"golang-base/internal/pkg/metrics"
	pkgstorage "golang-base/internal/pkg/storage"

	"github.com/gofiber/fiber/v3"
)

// New creates and configures the Fiber application with all routes.
func New(cfg *config.Config) *fiber.App {
	if cfg == nil {
		cfg = &config.Config{
			AppService:    "golang-base",
			AppVersion:    "dev",
			LogLevel:      "info",
			TracingSample: 1,
		}
	}

	// Setup structured logger
	logger.Setup(cfg.LogLevel, cfg.LogPretty)

	serviceName := "golang-base"
	if cfg.AppService != "" {
		serviceName = cfg.AppService
	}

	bodyLimit := 10 * 1024 * 1024
	if cfg.MaxUploadSize > 0 {
		bodyLimit = int(cfg.MaxUploadSize)
	}

	// Proxy trust
	fiberCfg := fiber.Config{
		AppName:        serviceName,
		ErrorHandler:   customErrorHandler,
		ReadBufferSize: 32 * 1024,
		BodyLimit:      bodyLimit,
	}
	if len(cfg.TrustedProxies) > 0 {
		fiberCfg.TrustProxy = true
		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{
			Proxies: cfg.TrustedProxies,
		}
		fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
		fiberCfg.EnableIPValidation = true
		slog.Info("proxy trust enabled", "cidrs", cfg.TrustedProxies, "header", fiber.HeaderXForwardedFor)
	} else {
		slog.Warn("proxy trust disabled: log IP will always be the direct TCP peer. Set TRUSTED_PROXIES behind a reverse proxy.")
	}

	app := fiber.New(fiberCfg)

	// Prometheus collectors
	m := metrics.New(serviceName, cfg.AppVersion)
	m.RegisterDB(database.SQL, serviceName)
	m.RegisterRedis(database.RedisPoolStats)

	// Middleware
	middleware.SetupMiddleware(app, cfg, m)

	// Scrape endpoint
	metricsEnabled := cfg.MetricsToken != ""
	if metricsEnabled {
		app.Get(middleware.MetricsPath, middleware.MetricsAuth(cfg.MetricsToken), m.Handler())
	} else {
		slog.Warn("metrics endpoint disabled because METRICS_TOKEN is not set")
	}

	// Storage
	storageProvider, _ := pkgstorage.NewProviderFromAppConfig(cfg)

	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		status := fiber.Map{
			"status":  "healthy",
			"service": serviceName,
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
		if storageProvider != nil {
			if err := storageProvider.Ping(c.Context()); err != nil {
				status["storage"] = "disconnected"
			} else {
				status["storage"] = "connected"
			}
		} else {
			status["storage"] = "disabled"
		}
		return c.JSON(status)
	})

	// Modules
	apiGroup := app.Group("/api/v1")
	docs.New().Register(apiGroup)
	auth.New().Register(apiGroup)
	user.New().Register(apiGroup)
	storage.New(cfg).Register(apiGroup)

	return app
}

// customErrorHandler provides consistent error responses
func customErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		if e.Message != "" {
			message = e.Message
		}
	}

	switch code {
	case fiber.StatusBadRequest:
		message = "Bad request"
	case fiber.StatusUnauthorized:
		message = "Unauthorized"
	case fiber.StatusForbidden:
		message = "Forbidden"
	case fiber.StatusNotFound:
		message = "Resource not found"
	case fiber.StatusConflict:
		message = "Resource conflict"
	case fiber.StatusUnprocessableEntity:
		message = "Unprocessable entity"
	case fiber.StatusRequestEntityTooLarge:
		message = "Request entity too large"
	case fiber.StatusTooManyRequests:
		message = "Too many requests"
	}

	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": message,
	})
}
