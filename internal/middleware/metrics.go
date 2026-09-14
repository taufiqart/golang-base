package middleware

import (
	"crypto/subtle"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"golang-base/internal/pkg/metrics"
)

// MetricsPath is where the Prometheus scrape endpoint is mounted.
const MetricsPath = "/metrics"

// bearerPrefix is the only accepted Authorization scheme for scrapes.
const bearerPrefix = "Bearer "

// Metrics records RED (rate, errors, duration) metrics for every routed request.
func Metrics(m *metrics.Metrics) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Scrape traffic is not user traffic: counting it makes the request rate
		// self-referential and, on a quiet service, dominates the metric.
		if c.Path() == MetricsPath {
			return c.Next()
		}

		start := time.Now()
		m.RequestsInFlight.Inc()

		// Deferred so a panic further down cannot leave the gauge permanently
		// elevated. The recover middleware sits outside this one, so without this
		// the in-flight count would drift upward on every crash.
		defer m.RequestsInFlight.Dec()

		observed := false
		defer func() {
			if !observed {
				// The panic still cost the client a 500; hide nothing from RED.
				m.ObserveRequest(routeLabel(c), c.Method(), fiber.StatusInternalServerError, time.Since(start).Seconds())
			}
		}()

		err := c.Next()
		m.ObserveRequest(routeLabel(c), c.Method(), effectiveStatus(c, err), time.Since(start).Seconds())
		observed = true

		return err
	}
}

// routeLabel returns the low-cardinality name for a request.
//
// Fiber's Route() falls back to a synthetic route whose Path is the raw request
// URL, so it is only trustworthy once the router reports a match. Labelling
// unmatched traffic by URL would hand a scanner unlimited time series.
func routeLabel(c fiber.Ctx) string {
	if c.Matched() {
		return c.Route().Path
	}
	return metrics.UnmatchedRoute
}

// MetricsAuth guards the Prometheus exposition endpoint.
//
// Register it before the metrics handler, e.g.
// app.Get(MetricsPath, middleware.MetricsAuth(token), m.Handler()).
//
// The endpoint leaks internal route names, traffic volumes and resource usage,
// so it is never anonymous. An unconfigured token answers 404 instead of
// serving metrics, so a missing secret cannot publish them by accident.
func MetricsAuth(token string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if token == "" {
			return fiber.ErrNotFound
		}

		header := c.Get(fiber.HeaderAuthorization)
		if len(header) <= len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
			return fiber.NewError(fiber.StatusUnauthorized, "missing metrics bearer token")
		}

		// Constant-time compare: the token is a secret, and a timing oracle here
		// would let an attacker recover it byte by byte.
		if subtle.ConstantTimeCompare([]byte(header[len(bearerPrefix):]), []byte(token)) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid metrics bearer token")
		}

		return c.Next()
	}
}
