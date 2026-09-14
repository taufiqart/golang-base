package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/logger"
)

func TestRequestLoggerRecordsRequestMetadata(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ok", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	record := findRecord(t, parseLogRecords(t, buf), "http request")
	assert.Equal(t, "GET", record["method"])
	assert.Equal(t, "/ok", record["path"])
	assert.Equal(t, float64(200), record["status"])
	assert.Equal(t, "INFO", record["level"])
	assert.Contains(t, record, "latency_ms")
	assert.Contains(t, record, "ip")
	assert.NotEmpty(t, record[logger.RequestIDKey], "request log must carry a request id")
}

func TestRequestLoggerWritesRequestIDExactlyOnce(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ok", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	line := strings.TrimSpace(buf.String())
	require.NotEmpty(t, line)
	assert.Equal(t, 1, strings.Count(line, `"`+logger.RequestIDKey+`"`),
		"duplicate JSON keys make log records ambiguous to parse: %s", line)
}

func TestRequestLoggerRecordsUserAgent(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	req := httptest.NewRequest(fiber.MethodGet, "/ok", nil)
	req.Header.Set("User-Agent", "integration-probe/1.0")

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	record := findRecord(t, parseLogRecords(t, buf), "http request")
	assert.Equal(t, "integration-probe/1.0", record["user_agent"])
}

func TestRequestLoggerSeverityFollowsStatus(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string
	}{
		{name: "success", path: "/ok", wantStatus: 200, wantLevel: "INFO"},
		{name: "client error", path: "/bad-request", wantStatus: 400, wantLevel: "WARN"},
		{name: "server error", path: "/boom", wantStatus: 500, wantLevel: "ERROR"},
		{name: "not found", path: "/missing", wantStatus: 404, wantLevel: "WARN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newLogApp()
			buf := captureSlogOutput(t)

			res, err := app.Test(httptest.NewRequest(fiber.MethodGet, tc.path, nil), fiber.TestConfig{Timeout: 10000})
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, res.StatusCode)
			assert.NoError(t, res.Body.Close())

			record := findRecord(t, parseLogRecords(t, buf), "http request")
			assert.Equal(t, tc.wantLevel, record["level"])
			assert.Equal(t, float64(tc.wantStatus), record["status"])
		})
	}
}

func TestRequestLoggerRecordsHandlerError(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/boom", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	record := findRecord(t, parseLogRecords(t, buf), "http request")
	assert.Equal(t, "kaboom", record["error"])
}

func TestRequestLoggerDoesNotSwallowHandlerError(t *testing.T) {
	app := newLogApp()
	captureSlogOutput(t)

	app.Get("/handler-error", func(c fiber.Ctx) error {
		return fiber.ErrTeapot
	})

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/handler-error", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	assert.Equal(t, fiber.StatusTeapot, res.StatusCode, "error handler must still run after logging")
}

func TestHandlerLogsInheritRequestID(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/handler-logs", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, res.StatusCode)
	assert.NoError(t, res.Body.Close())

	records := parseLogRecords(t, buf)
	handler := findRecord(t, records, "handler work done")
	request := findRecord(t, records, "http request")

	assert.Equal(t, "u-42", handler["user_id"])
	assert.NotEmpty(t, handler[logger.RequestIDKey], "handler logs must carry the request id")
	assert.Equal(t, request[logger.RequestIDKey], handler[logger.RequestIDKey],
		"request log and handler log must share one request id")
}
