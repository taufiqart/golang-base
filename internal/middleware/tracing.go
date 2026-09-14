package middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"golang-base/internal/pkg/logger"
	"golang-base/internal/pkg/tracing"
)

// OpenTelemetry semantic convention attribute keys used on HTTP server spans.
const (
	attrHTTPMethod    = "http.request.method"
	attrURLPath       = "url.path"
	attrHTTPRoute     = "http.route"
	attrHTTPStatus    = "http.response.status_code"
	attrNetworkPeerIP = "network.peer.address"

	// attrRequestID reuses the logger key so the span and the log line label the
	// correlation id identically.
	attrRequestID = logger.RequestIDKey
)

// Tracing starts one server span per request, continuing an inbound W3C trace
// when a caller sends a traceparent header.
//
// It also copies trace_id and span_id onto the request-scoped logger, which is
// what makes logs and traces searchable together. Register it after RequestID
// (so the request id exists) and before RequestLogger (so request records carry
// the trace ids).
//
// When tracing is disabled the global provider is OTel's no-op implementation,
// so this middleware costs nothing and needs no enabled flag of its own.
func Tracing() fiber.Handler {
	tracer := otel.Tracer(tracing.InstrumentationScope)

	return func(c fiber.Ctx) error {
		ctx := otel.GetTextMapPropagator().Extract(c.Context(), headerCarrier{c})

		// The matched route is unknown until the handler runs, so the span starts
		// under a bounded name and is renamed afterwards. Naming it from the raw
		// URL would give every distinct path its own span name.
		ctx, span := tracer.Start(ctx, "HTTP "+c.Method(),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String(attrHTTPMethod, c.Method()),
				attribute.String(attrURLPath, c.Path()),
				attribute.String(attrNetworkPeerIP, c.IP()),
			),
		)
		defer span.End()

		if sc := span.SpanContext(); sc.IsValid() {
			ctx = logger.WithLogger(ctx, logger.FromContext(ctx).With(
				logger.TraceIDKey, sc.TraceID().String(),
				logger.SpanIDKey, sc.SpanID().String(),
			))
		}
		if rid := RequestIDFromContext(c); rid != "" {
			span.SetAttributes(attribute.String(attrRequestID, rid))
		}

		c.SetContext(ctx)

		err := c.Next()
		status := effectiveStatus(c, err)

		if c.Matched() {
			route := c.Route().Path
			span.SetName(c.Method() + " " + route)
			span.SetAttributes(attribute.String(attrHTTPRoute, route))
		}

		span.SetAttributes(attribute.Int(attrHTTPStatus, status))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= fiber.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(status))
		}

		return err
	}
}

// headerCarrier adapts a Fiber request to OTel's TextMapCarrier. It reads and
// writes headers in place instead of copying the whole header set per request.
type headerCarrier struct{ c fiber.Ctx }

// Get returns the value of the given header.
func (h headerCarrier) Get(key string) string { return h.c.Get(key) }

// Set writes a response header; it is only used for injection, which server
// spans do not perform.
func (h headerCarrier) Set(key, value string) { h.c.Response().Header.Set(key, value) }

// Keys lists the request header names.
func (h headerCarrier) Keys() []string {
	var keys []string
	h.c.Request().Header.VisitAll(func(k, v []byte) {
		keys = append(keys, string(k))
	})
	return keys
}
