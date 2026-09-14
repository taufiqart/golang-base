package middleware

import (
	"time"

	"golang-base/config"
	"golang-base/internal/pkg/metrics"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// SetupMiddleware configures all global middleware for the Fiber app.
func SetupMiddleware(app *fiber.App, cfg *config.Config, m *metrics.Metrics) {
	app.Use(recover.New())

	// Measured first so latency covers everything the client waits on.
	if m != nil {
		app.Use(Metrics(m))
	}

	app.Use(RequestID())
	app.Use(Tracing())
	app.Use(RequestLogger())

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	// Rate Limiter
	maxReqs := 100
	expMins := time.Minute
	if cfg != nil {
		if cfg.RateLimitMax > 0 {
			maxReqs = cfg.RateLimitMax
		}
		if cfg.RateLimitExpMin > 0 {
			expMins = time.Duration(cfg.RateLimitExpMin) * time.Minute
		}
	}

	app.Use(limiter.New(limiter.Config{
		Max:        maxReqs,
		Expiration: expMins,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "Too many requests, please try again later.",
			})
		},
	}))
}
