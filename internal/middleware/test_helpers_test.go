package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"golang-base/internal/pkg/logger"
)

// captureSlogOutput points the global slog default at an in-memory JSON buffer
// and restores the previous logger when the test ends.
func captureSlogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return buf
}

func parseLogRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record), "line: %s", line)
		records = append(records, record)
	}

	return records
}

func findRecord(t *testing.T, records []map[string]any, msg string) map[string]any {
	t.Helper()

	for _, r := range records {
		if r["msg"] == msg {
			return r
		}
	}
	require.FailNowf(t, "record not found", "no log record with msg %q in %v", msg, records)
	return nil
}

// newLogApp wires the correlation ID and request logger middleware over four
// routes covering success, client error, server error and handler logging.
func newLogApp() *fiber.App {
	app := fiber.New()
	app.Use(RequestID())
	app.Use(RequestLogger())

	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/bad-request", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad input")
	})
	app.Get("/boom", func(c fiber.Ctx) error {
		return errors.New("kaboom")
	})
	app.Get("/handler-logs", func(c fiber.Ctx) error {
		logger.FromContext(c.Context()).Info("handler work done", "user_id", "u-42")
		return c.SendString("logged")
	})

	return app
}
