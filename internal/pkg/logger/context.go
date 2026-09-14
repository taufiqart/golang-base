package logger

import (
	"context"
	"log/slog"
)

// Attribute keys used for correlation. The HTTP middleware writes request_id,
// trace_id and span_id onto the request-scoped logger so a log line and its
// trace can be found from any one of the three.
const (
	RequestIDKey = "request_id"
	TraceIDKey   = "trace_id"
	SpanIDKey    = "span_id"
)

// ctxKey is the context key under which a request-scoped logger is stored.
type ctxKey struct{}

// WithLogger stores l in ctx so downstream layers can retrieve a logger that
// already carries request-scoped attributes.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil || l == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the request-scoped logger stored in ctx.
//
// When no logger was stored it falls back to slog.Default(), so callers never
// need a nil check.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// WithRequestID stores the correlation ID in ctx and returns a logger that
// emits it as a request_id attribute on every record. The derived context
// carries the logger so FromContext finds it further down the stack.
func WithRequestID(ctx context.Context, requestID string) (context.Context, *slog.Logger) {
	if ctx == nil {
		ctx = context.Background()
	}
	l := slog.Default()
	if requestID != "" {
		l = l.With(RequestIDKey, requestID)
	}
	return WithLogger(ctx, l), l
}
