package middleware

import (
	"time"

	"golang-base/config"
	"golang-base/internal/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupMiddleware configures all global middleware for the Fiber app
func SetupMiddleware(app *fiber.App, cfg *config.Config) {
	// Recover from panics to prevent server crashes
	app.Use(recover.New())

	// HTTP request logging
	app.Use(fiberLogger.New(fiberLogger.Config{
		Format:     "[${time}] ${status} - ${latency} | ${ip} | ${method} ${path} ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Asia/Jakarta",
		Output:     logger.GetOutput(),
	}))

	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Rate Limiter to prevent DDoS and Brute Force attacks
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
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "Too many requests, please try again later.",
			})
		},
	}))
}
