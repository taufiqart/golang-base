package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{name: "debug", input: "debug", want: slog.LevelDebug},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "warn", input: "warn", want: slog.LevelWarn},
		{name: "warning alias", input: "warning", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{name: "uppercase", input: "INFO", want: slog.LevelInfo},
		{name: "padded", input: "  WaRn  ", want: slog.LevelWarn},
		{name: "unknown", input: "trace", want: DefaultLevel, wantErr: true},
		{name: "empty", input: "", want: DefaultLevel, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLevel(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLevelName(t *testing.T) {
	assert.Equal(t, "debug", LevelName(slog.LevelDebug))
	assert.Equal(t, "info", LevelName(slog.LevelInfo))
	assert.Equal(t, "warn", LevelName(slog.LevelWarn))
	assert.Equal(t, "error", LevelName(slog.LevelError))
	assert.Equal(t, "error", LevelName(slog.LevelError+4))
}

func setupTestDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() {
		SetDir("")
		Setup("info")
	})

	return dir
}

func readLogFile(t *testing.T, dir, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}

	return string(raw)
}

func logAllLevels() {
	slog.Debug("debug message")
	slog.Info("info message")
	slog.Warn("warn message")
	slog.Error("error message")
}

func TestSetupCreatesPerLevelFiles(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug")

	logAllLevels()

	tests := []struct {
		file     string
		contains string
		excludes []string
	}{
		{file: FileApp, contains: "debug message", excludes: nil},
		{file: FileDebug, contains: "debug message", excludes: []string{"info message", "warn message", "error message"}},
		{file: FileInfo, contains: "info message", excludes: []string{"debug message", "warn message", "error message"}},
		{file: FileWarn, contains: "warn message", excludes: []string{"debug message", "info message", "error message"}},
		{file: FileError, contains: "error message", excludes: []string{"debug message", "info message", "warn message"}},
	}

	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			content := readLogFile(t, dir, tc.file)
			require.NotEmpty(t, content, "%s should exist and be written", tc.file)
			assert.Contains(t, content, tc.contains)
			for _, excluded := range tc.excludes {
				assert.NotContains(t, content, excluded)
			}
		})
	}
}

func TestAppLogContainsEveryLevel(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug")

	logAllLevels()

	content := readLogFile(t, dir, FileApp)
	for _, want := range []string{"debug message", "info message", "warn message", "error message"} {
		assert.Contains(t, content, want)
	}
}

func TestSetupLevelFiltersRecords(t *testing.T) {
	dir := setupTestDir(t)
	Setup("warn")

	logAllLevels()

	assert.Empty(t, readLogFile(t, dir, FileDebug), "debug.log must stay empty below warn level")
	assert.Empty(t, readLogFile(t, dir, FileInfo), "info.log must stay empty below warn level")
	assert.Contains(t, readLogFile(t, dir, FileWarn), "warn message")
	assert.Contains(t, readLogFile(t, dir, FileError), "error message")

	appContent := readLogFile(t, dir, FileApp)
	assert.NotContains(t, appContent, "info message")
	assert.Contains(t, appContent, "warn message")
}

func TestSetupRejectsUnknownLevel(t *testing.T) {
	dir := setupTestDir(t)

	Setup("not-a-level")

	assert.Equal(t, DefaultLevel, Level())

	slog.Info("still logged")
	assert.Contains(t, readLogFile(t, dir, FileApp), "still logged")
}

func TestSetupRecordsEntriesAsJSON(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug")

	slog.Info("structured", "user_id", "u-1", "count", 3)

	line := strings.TrimSpace(readLogFile(t, dir, FileApp))
	require.NotEmpty(t, line)

	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &record))
	assert.Equal(t, "INFO", record["level"])
	assert.Equal(t, "structured", record["msg"])
	assert.Equal(t, "u-1", record["user_id"])
	assert.Equal(t, float64(3), record["count"])
}

func TestGetOutputFallsBackToStdout(t *testing.T) {
	mu.Lock()
	previous := appOutput
	appOutput = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		appOutput = previous
		mu.Unlock()
	})

	assert.Equal(t, os.Stdout, GetOutput())
}

func TestGetOutputAfterSetup(t *testing.T) {
	setupTestDir(t)
	Setup("info")

	assert.NotNil(t, GetOutput())
	assert.NotEqual(t, os.Stdout, GetOutput())
}

func TestWithRequestID(t *testing.T) {
	dir := setupTestDir(t)
	Setup("info")

	ctx, l := WithRequestID(context.Background(), "req-123")
	l.Error("something broke", "error", "boom")

	assert.NotNil(t, FromContext(ctx))

	content := readLogFile(t, dir, FileError)
	require.NotEmpty(t, content)

	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(content)), &record))
	assert.Equal(t, "req-123", record[RequestIDKey])
	assert.Equal(t, "something broke", record["msg"])
	assert.Equal(t, "boom", record["error"])
}

func TestWithRequestIDEmptyID(t *testing.T) {
	setupTestDir(t)
	Setup("info")

	ctx, _ := WithRequestID(context.Background(), "")

	assert.Equal(t, slog.Default(), FromContext(ctx))
}

func TestFromContextDefaults(t *testing.T) {
	assert.Equal(t, slog.Default(), FromContext(context.Background()))
	assert.Equal(t, slog.Default(), FromContext(nil))

	ctx := context.WithValue(context.Background(), ctxKey{}, "not-a-logger")
	assert.Equal(t, slog.Default(), FromContext(ctx))
}

func TestWithLoggerNilSafe(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, ctx, WithLogger(ctx, nil))
	assert.Nil(t, WithLogger(nil, slog.Default()))
}

func TestRequestScopedLoggerPropagatesAttributes(t *testing.T) {
	dir := setupTestDir(t)
	Setup("info")

	_, l := WithRequestID(context.Background(), "req-abc")

	l.With("layer", "service").Info("nested call")

	content := readLogFile(t, dir, FileInfo)
	assert.Contains(t, content, "req-abc")
	assert.Contains(t, content, "nested call")
	assert.Contains(t, content, "service")
}
