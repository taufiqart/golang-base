package middleware

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"golang-base/internal/pkg/logger"
)

// RequestLogger emits one structured JSON log record per HTTP request.
//
// The record inherits request_id (and trace_id when tracing is on) from the
// request-scoped logger installed by the upstream middleware, so request lines
// and application logs join on the same identifiers. Records go through slog,
// which also writes them to logs/app.log and logs/<level>.log.
func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		handlerErr := c.Next()

		status := effectiveStatus(c, handlerErr)

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			// IP is the client address: TCP peer when no proxy is trusted,
			// otherwise the X-Forwarded-For value from a trusted proxy.
			"ip", c.IP(),
		}

		if ua := c.Get("User-Agent"); ua != "" {
			attrs = append(attrs, "user_agent", ua)
		}
		if handlerErr != nil {
			attrs = append(attrs, "error", handlerErr.Error())
		}

		l := logger.FromContext(c.Context())
		switch {
		case status >= fiber.StatusInternalServerError:
			l.Error("http request", attrs...)
		case status >= fiber.StatusBadRequest:
			l.Warn("http request", attrs...)
		default:
			l.Info("http request", attrs...)
		}

		return handlerErr
	}
}

// effectiveStatus reports the status the client will see.
//
// Fiber's ErrorHandler runs after this middleware unwinds, so a returned error
// has not been converted to a response status yet. Reading only
// c.Response().StatusCode() would log a failed request as a 200.
func effectiveStatus(c fiber.Ctx, handlerErr error) int {
	status := c.Response().StatusCode()
	if handlerErr == nil {
		return status
	}

	var fe *fiber.Error
	if errors.As(handlerErr, &fe) {
		return fe.Code
	}

	return fiber.StatusInternalServerError
}
