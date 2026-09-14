package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"golang-base/internal/pkg/logger"
)

// RequestIDHeader is the header used to exchange the request correlation ID.
const RequestIDHeader = fiber.HeaderXRequestID

// requestIDLocalKey is the Fiber locals key holding the correlation ID.
const requestIDLocalKey = "requestid"

// maxRequestIDLen caps an inbound correlation ID. Longer values are treated as
// hostile or malformed input and replaced with a generated ID, which keeps log
// records bounded.
const maxRequestIDLen = 256

// RequestID assigns every request a unique correlation ID.
//
// It reuses an incoming X-Request-ID header when the client supplies a sane one
// and generates a UUIDv7 otherwise, echoes the ID in the response header,
// stores it in Fiber locals, and installs a request-scoped slog logger carrying
// the request_id attribute on the request context.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := sanitizeRequestID(c.Get(RequestIDHeader))

		c.Set(RequestIDHeader, id)
		c.Locals(requestIDLocalKey, id)

		ctx, _ := logger.WithRequestID(c.Context(), id)
		c.SetContext(ctx)

		return c.Next()
	}
}

// sanitizeRequestID keeps the inbound id only when it is a short printable
// ASCII token, otherwise it returns a freshly generated UUIDv7.
func sanitizeRequestID(inbound string) string {
	id := strings.TrimSpace(inbound)
	if id == "" || len(id) > maxRequestIDLen {
		return newRequestID()
	}
	for i := 0; i < len(id); i++ {
		if id[i] < 0x20 || id[i] == 0x7f {
			return newRequestID()
		}
	}
	return id
}

func newRequestID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// RequestIDFromContext returns the correlation ID of the current request, or an
// empty string when the RequestID middleware did not run.
func RequestIDFromContext(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	if id, ok := c.Locals(requestIDLocalKey).(string); ok {
		return id
	}
	return ""
}
