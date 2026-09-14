package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupPrettyWritesTextLines(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug", true)

	slog.Warn("pretty warning", "user_id", "u-7")

	raw, err := os.ReadFile(filepath.Join(dir, FileWarn))
	require.NoError(t, err)

	line := strings.TrimSpace(string(raw))
	assert.Contains(t, line, "level=WARN")
	// slog quotes values that contain spaces.
	assert.Contains(t, line, `msg="pretty warning"`)
	assert.Contains(t, line, "user_id=u-7")
	assert.NotContains(t, line, "{", "pretty mode must not emit JSON")
}

func TestSetupPrettyStillRoutesByLevel(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug", true)

	slog.Info("pretty info")
	slog.Error("pretty error")

	infoRaw, err := os.ReadFile(filepath.Join(dir, FileInfo))
	require.NoError(t, err)
	assert.Contains(t, string(infoRaw), "pretty info")
	assert.NotContains(t, string(infoRaw), "pretty error")

	errorRaw, err := os.ReadFile(filepath.Join(dir, FileError))
	require.NoError(t, err)
	assert.Contains(t, string(errorRaw), "pretty error")
	assert.NotContains(t, string(errorRaw), "pretty info")
}

func TestSetupPrettyHasNoAnsiCodes(t *testing.T) {
	dir := setupTestDir(t)
	Setup("debug", true)

	slog.Error("colour check")

	raw, err := os.ReadFile(filepath.Join(dir, FileError))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "\x1b[", "escape codes would corrupt the log files")
}

func TestSetupSwitchesBetweenFormats(t *testing.T) {
	dir := setupTestDir(t)

	Setup("info", false)
	slog.Info("json round")
	assert.Contains(t, string(mustRead(t, filepath.Join(dir, FileInfo))), `"msg":"json round"`)

	Setup("info", true)
	slog.Info("text round")
	content := string(mustRead(t, filepath.Join(dir, FileInfo)))
	assert.Contains(t, content, `msg="text round"`)
}

// TestSetupDoesNotLeakFileHandles pins the behaviour that makes the repeated
// Setup from main and app.New safe: no-op on identical arguments, and the old
// handles closed whenever the logger really is rebuilt.
func TestSetupDoesNotLeakFileHandles(t *testing.T) {
	dir := setupTestDir(t)
	Setup("info", false)

	assert.Equal(t, 1, appHandleCount(t, dir))

	Setup("info", false)
	Setup("info", false)
	assert.Equal(t, 1, appHandleCount(t, dir), "unchanged setup must not open a second handle")

	for _, level := range []string{"debug", "warn", "error", "info"} {
		Setup(level, false)
	}
	Setup("info", true)
	assert.Equal(t, 1, appHandleCount(t, dir), "every rebuild must close the handle it replaces")

	// Still functioning after all that reconfiguration.
	slog.Info("still logging")
	assert.Contains(t, string(mustRead(t, filepath.Join(dir, FileInfo))), `msg="still logging"`)
}

// appHandleCount counts open descriptors pointing at the app log file, which is
// how a leaked handle from a previous Setup would show up.
func appHandleCount(t *testing.T, dir string) int {
	t.Helper()

	target := filepath.Join(dir, FileApp)

	matches := 0
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("cannot inspect /proc/self/fd on this platform: %v", err)
	}

	for _, entry := range entries {
		link, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			continue
		}
		if filepath.Clean(link) == filepath.Clean(target) {
			matches++
		}
	}

	return matches
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	return raw
}
