package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/logger"
)

func TestRequestIDGeneratesAndEchoesID(t *testing.T) {
	app := newLogApp()

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ok", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)

	id := res.Header.Get(RequestIDHeader)
	assert.NotEmpty(t, id, "response must carry a generated X-Request-ID")
	parsed, parseErr := uuid.Parse(id)
	require.NoError(t, parseErr, "generated id must be a valid UUID")
	assert.Equal(t, uuid.Version(7), parsed.Version(), "generated id must be UUIDv7")
	assert.NoError(t, res.Body.Close())
}

func TestRequestIDGeneratesUniqueIDsPerRequest(t *testing.T) {
	app := newLogApp()

	seen := map[string]bool{}
	for range 5 {
		res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/ok", nil), fiber.TestConfig{Timeout: 10000})
		require.NoError(t, err)
		id := res.Header.Get(RequestIDHeader)
		require.NotEmpty(t, id)
		assert.False(t, seen[id], "request id %q must be unique", id)
		seen[id] = true
		assert.NoError(t, res.Body.Close())
	}
}

func TestRequestIDReusesIncomingHeader(t *testing.T) {
	app := newLogApp()
	buf := captureSlogOutput(t)

	req := httptest.NewRequest(fiber.MethodGet, "/ok", nil)
	req.Header.Set(RequestIDHeader, "upstream-trace-1")

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.Equal(t, "upstream-trace-1", res.Header.Get(RequestIDHeader))
	assert.NoError(t, res.Body.Close())

	record := findRecord(t, parseLogRecords(t, buf), "http request")
	assert.Equal(t, "upstream-trace-1", record[logger.RequestIDKey])
}

func TestRequestIDRejectsOversizedHeaderAndGenerates(t *testing.T) {
	app := newLogApp()

	oversized := strings.Repeat("x", maxRequestIDLen+1)
	req := httptest.NewRequest(fiber.MethodGet, "/ok", nil)
	req.Header.Set(RequestIDHeader, oversized)

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)

	id := res.Header.Get(RequestIDHeader)
	assert.NotEmpty(t, id)
	assert.NotEqual(t, oversized, id, "invalid inbound id must be replaced")
	assert.NoError(t, res.Body.Close())
}

func TestRequestIDRejectsControlCharactersInHeader(t *testing.T) {
	tests := []struct {
		name    string
		inbound string
	}{
		{name: "newline injection", inbound: "abc\ndef"},
		{name: "carriage return", inbound: "abc\r\nevil: yes"},
		{name: "tab", inbound: "abc\tdef"},
		{name: "blank", inbound: "   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newLogApp()

			req := httptest.NewRequest(fiber.MethodGet, "/ok", nil)
			req.Header.Set(RequestIDHeader, tc.inbound)

			res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
			require.NoError(t, err)

			id := res.Header.Get(RequestIDHeader)
			assert.NotEqual(t, tc.inbound, id, "unsafe inbound id must be replaced")
			assert.NotEmpty(t, id)
			assert.NoError(t, res.Body.Close())
		})
	}
}

func TestSanitizeRequestID(t *testing.T) {
	tests := []struct {
		name    string
		inbound string
		want    string
		reuse   bool
	}{
		{name: "reuse short token", inbound: "trace-abc", want: "trace-abc", reuse: true},
		{name: "trim surrounding space", inbound: "  trace-abc  ", want: "trace-abc", reuse: true},
		{name: "empty generates", inbound: "", reuse: false},
		{name: "whitespace only generates", inbound: " \t ", reuse: false},
		{name: "exact max length reused", inbound: strings.Repeat("a", maxRequestIDLen), want: strings.Repeat("a", maxRequestIDLen), reuse: true},
		{name: "over max length regenerated", inbound: strings.Repeat("a", maxRequestIDLen+1), reuse: false},
		{name: "control byte rejected", inbound: "ab\x00cd", reuse: false},
		{name: "del byte rejected", inbound: "abc\x7f", reuse: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeRequestID(tc.inbound)
			if tc.reuse {
				assert.Equal(t, tc.want, got)
				return
			}
			assert.NotEqual(t, strings.TrimSpace(tc.inbound), got)
			parsed, err := uuid.Parse(got)
			require.NoError(t, err)
			assert.Equal(t, uuid.Version(7), parsed.Version())
		})
	}
}

func TestRequestIDFromContext(t *testing.T) {
	app := newLogApp()

	var observed string
	app.Get("/capture", func(c fiber.Ctx) error {
		observed = RequestIDFromContext(c)
		return c.SendString("done")
	})

	res, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/capture", nil), fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	assert.NoError(t, res.Body.Close())

	assert.NotEmpty(t, observed)
	assert.Equal(t, res.Header.Get(RequestIDHeader), observed)
	assert.Empty(t, RequestIDFromContext(nil))
}
