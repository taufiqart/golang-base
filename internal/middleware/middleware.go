package middleware

import (
	"time"

	"golang-base/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

func SetupMiddleware(app *fiber.App, cfg *config.Config) {
	app.Use(recover.New())
	app.Use(RequestID())
	app.Use(RequestLogger())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
	maxReqs := 100
	expMins := time.Minute
	if cfg != nil {
		if cfg.RateLimitMax > 0 { maxReqs = cfg.RateLimitMax }
		if cfg.RateLimitExpMin > 0 { expMins = time.Duration(cfg.RateLimitExpMin) * time.Minute }
	}
	app.Use(limiter.New(limiter.Config{
		Max:        maxReqs,
		Expiration: expMins,
		KeyGenerator: func(c fiber.Ctx) string { return c.IP() },
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"message": "Too many requests"})
		},
	}))
}
