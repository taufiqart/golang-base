package middleware

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"golang-base/internal/pkg/logger"
)

// installSpanRecorder swaps the global tracer provider for one backed by an
// in-memory recorder and restores the previous provider afterwards.
func installSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	return recorder
}

func newTracingApp() *fiber.App {
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Tracing())
	app.Use(RequestLogger())
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/users/:id", func(c fiber.Ctx) error { return c.SendString("user") })
	app.Get("/boom", func(c fiber.Ctx) error { return errors.New("kaboom") })
	return app
}

func doRequest(t *testing.T, app *fiber.App, path string, headers map[string]string) {
	t.Helper()

	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())
}

func attrValue(span sdktrace.ReadOnlySpan, key string) (attribute.KeyValue, bool) {
	for _, kv := range span.Attributes() {
		if string(kv.Key) == key {
			return kv, true
		}
	}
	return attribute.KeyValue{}, false
}

func TestTracingCreatesServerSpanWithRouteTemplate(t *testing.T) {
	recorder := installSpanRecorder(t)

	doRequest(t, newTracingApp(), "/users/42", nil)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "GET /users/:id", span.Name(),
		"span name must use the route template, not the raw URL")
	assert.Equal(t, trace.SpanKindServer, span.SpanKind())

	assert.True(t, span.SpanContext().IsValid())
	assert.False(t, span.SpanContext().IsRemote(), "a fresh trace starts locally")

	routeAttr, ok := attrValue(span, "http.route")
	require.True(t, ok, "span must carry http.route")
	assert.Equal(t, "/users/:id", routeAttr.Value.AsString())

	statusAttr, ok := attrValue(span, "http.response.status_code")
	require.True(t, ok, "span must carry the response status")
	assert.Equal(t, int64(200), statusAttr.Value.AsInt64())

	methodAttr, ok := attrValue(span, "http.request.method")
	require.True(t, ok)
	assert.Equal(t, "GET", methodAttr.Value.AsString())
}

func TestTracingContinuesInboundTrace(t *testing.T) {
	recorder := installSpanRecorder(t)

	const (
		inboundTraceID = "0af7651916cd43dd8448eb211c80319c"
		inboundSpanID  = "b7ad6b7169203331"
	)
	traceparent := "00-" + inboundTraceID + "-" + inboundSpanID + "-01"

	doRequest(t, newTracingApp(), "/ok", map[string]string{"traceparent": traceparent})

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	sc := spans[0].SpanContext()

	assert.Equal(t, inboundTraceID, sc.TraceID().String(),
		"the service must stay inside the caller's trace")
	assert.Equal(t, inboundSpanID, spans[0].Parent().SpanID().String(),
		"the inbound span must be recorded as the parent")
	assert.True(t, sc.IsSampled(), "the sampled inbound flag must be honoured")
}

func TestTracingStartsNewTraceWithoutInboundHeaders(t *testing.T) {
	recorder := installSpanRecorder(t)

	doRequest(t, newTracingApp(), "/ok", nil)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.False(t, spans[0].Parent().IsValid(), "no inbound header means a root span")
}

func TestTracingDoesNotPutRawPathInSpanName(t *testing.T) {
	recorder := installSpanRecorder(t)

	doRequest(t, newTracingApp(), "/probe/9f8e7d6c5b4a", nil)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "HTTP GET", spans[0].Name(),
		"unmatched requests must keep the bounded default span name")
	assert.NotContains(t, spans[0].Name(), "9f8e7d6c5b4a")

	_, hasRoute := attrValue(spans[0], "http.route")
	assert.False(t, hasRoute, "http.route must be absent when no route matched")
}

func TestTracingMarksErrorSpans(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantCode   codes.Code
		wantStatus int
	}{
		{name: "handler error", path: "/boom", wantCode: codes.Error, wantStatus: 500},
		{name: "success", path: "/ok", wantCode: codes.Unset, wantStatus: 200},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := installSpanRecorder(t)

			doRequest(t, newTracingApp(), tc.path, nil)

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, tc.wantCode, spans[0].Status().Code)

			statusAttr, ok := attrValue(spans[0], "http.response.status_code")
			require.True(t, ok)
			assert.Equal(t, int64(tc.wantStatus), statusAttr.Value.AsInt64())
		})
	}
}

func TestTracingPropagatesTraceIDsToLogs(t *testing.T) {
	recorder := installSpanRecorder(t)
	buf := captureSlogOutput(t)

	app := newTracingApp()
	app.Get("/logs-trace", func(c fiber.Ctx) error {
		logger.FromContext(c.Context()).Info("work done")
		return c.SendString("ok")
	})

	doRequest(t, app, "/logs-trace", nil)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	wantTraceID := spans[0].SpanContext().TraceID().String()
	wantSpanID := spans[0].SpanContext().SpanID().String()

	records := parseLogRecords(t, buf)
	handler := findRecord(t, records, "work done")
	request := findRecord(t, records, "http request")

	for name, record := range map[string]map[string]any{"handler": handler, "request": request} {
		require.Equal(t, wantTraceID, record[logger.TraceIDKey], "%s log must carry the span trace id", name)
		require.Equal(t, wantSpanID, record[logger.SpanIDKey], "%s log must carry the span id", name)
		assert.NotEmpty(t, record[logger.RequestIDKey], "%s log must also carry the request id", name)
	}

	assert.Equal(t, request[logger.RequestIDKey], handler[logger.RequestIDKey])
}

func TestTracingIsFreeWhenDisabled(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(noop.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	buf := captureSlogOutput(t)

	doRequest(t, newTracingApp(), "/ok", nil)

	record := findRecord(t, parseLogRecords(t, buf), "http request")
	assert.NotContains(t, record, logger.TraceIDKey,
		"a no-op provider must not emit all-zero trace ids")
	assert.NotContains(t, record, logger.SpanIDKey)
	assert.NotEmpty(t, record[logger.RequestIDKey], "request correlation still works untraced")
}

func TestHeaderCarrierReadsRequestHeaders(t *testing.T) {
	app := fiber.New()

	var got string
	var missing string
	var keys []string

	app.Get("/carrier", func(c fiber.Ctx) error {
		// A fiber.Ctx is recycled after the request, so assertions must run here.
		carrier := headerCarrier{c}
		got = carrier.Get("X-Custom-Trace")
		missing = carrier.Get("X-Missing")
		keys = carrier.Keys()
		return c.SendString("ok")
	})

	doRequest(t, app, "/carrier", map[string]string{"X-Custom-Trace": "yes", "Accept": "application/json"})

	assert.Equal(t, "yes", got)
	assert.Equal(t, "", missing, "an absent header reads as the empty string")
	assert.NotEmpty(t, keys)

	joined := strings.ToLower(strings.Join(keys, ","))
	assert.Contains(t, joined, "x-custom-trace")
	assert.Contains(t, joined, "accept")
}

func TestHeaderCarrierSetsResponseHeader(t *testing.T) {
	app := fiber.New()

	var echoed string
	app.Get("/echo", func(c fiber.Ctx) error {
		headerCarrier{c}.Set("X-Trace-Echo", "value")
		echoed = c.GetRespHeader("X-Trace-Echo")
		return c.SendString("ok")
	})

	doRequest(t, app, "/echo", nil)
	assert.Equal(t, "value", echoed, "Set must write a response header for injection")
}
