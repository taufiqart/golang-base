// Package logger provides the application structured logger built on top of
// log/slog.
//
// Output files (all JSON Lines, one record per line):
//
//	logs/app.log    -- every record at or above the configured level
//	logs/debug.log  -- DEBUG records only
//	logs/info.log   -- INFO records only
//	logs/warn.log   -- WARN records only
//	logs/error.log  -- ERROR records only
//
// stdout always receives the same stream as logs/app.log so containers keep
// working unchanged.
package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Log directory and file names.
const (
	// Dir is the directory (relative to the process working directory) that
	// holds every log file.
	Dir = "logs"

	// FileApp holds every record at or above the configured level.
	FileApp = "app.log"
	// FileDebug holds DEBUG records only.
	FileDebug = "debug.log"
	// FileInfo holds INFO records only.
	FileInfo = "info.log"
	// FileWarn holds WARN records only.
	FileWarn = "warn.log"
	// FileError holds ERROR records only.
	FileError = "error.log"
)

// Level names accepted by ParseLevel and the LOG_LEVEL environment variable.
const (
	LevelDebugName = "debug"
	LevelInfoName  = "info"
	LevelWarnName  = "warn"
	LevelErrorName = "error"
)

// DefaultLevel is used when no level is configured.
const DefaultLevel = slog.LevelInfo

// orderedLevels keeps the per-level slice deterministic.
var orderedLevels = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

// levelFiles maps every concrete level to its dedicated file name.
var levelFiles = map[slog.Level]string{
	slog.LevelDebug: FileDebug,
	slog.LevelInfo:  FileInfo,
	slog.LevelWarn:  FileWarn,
	slog.LevelError: FileError,
}

var (
	mu        sync.RWMutex
	appOutput io.Writer
	openFiles []*os.File

	currentLevel = DefaultLevel
	initialised  bool
	activeDir    string
	activePretty bool

	// dir overrides Dir while it is non-empty. It exists so tests can redirect
	// log files to a temporary directory.
	dir string
)

func resolveDir() string {
	mu.RLock()
	defer mu.RUnlock()
	if dir != "" {
		return dir
	}
	return Dir
}

// Setup initializes the global slog logger.
//
// Every record is written to stdout plus the five log files described in the
// package documentation. The level argument accepts "debug", "info", "warn" and
// "error" (case-insensitive); an empty or unknown value falls back to
// DefaultLevel. Passing pretty=true renders human-readable lines instead of
// JSON, for local development.
//
// Repeated calls are a no-op while the level, format and directory are
// unchanged, so a process may call Setup from both main and its app bootstrap. A
// genuine change reopens the files, closes the previous handles, and swaps the
// default logger atomically.
func Setup(level string, pretty ...bool) {
	lvl, err := ParseLevel(level)
	if err != nil {
		log.Printf("logger: %v, falling back to %q", err, LevelName(DefaultLevel))
		lvl = DefaultLevel
	}

	shouldPretty := len(pretty) > 0 && pretty[0]
	logDir := resolveDir()

	mu.Lock()
	if initialised && currentLevel == lvl && activePretty == shouldPretty && activeDir == logDir {
		mu.Unlock()
		return
	}
	previous := openFiles
	mu.Unlock()

	handler, output, files, err := buildHandler(logDir, lvl, shouldPretty)
	if err != nil {
		log.Printf("logger: falling back to stdout only: %v", err)
		handler, output, files = nil, os.Stdout, nil
	}

	mu.Lock()
	currentLevel = lvl
	appOutput = output
	openFiles = files
	initialised = true
	activeDir = logDir
	activePretty = shouldPretty
	mu.Unlock()

	if handler == nil {
		slog.SetDefault(slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: lvl})))
	} else {
		slog.SetDefault(slog.New(handler))
	}

	for _, f := range previous {
		_ = f.Close()
	}
}

// Level returns the level the global logger was configured with.
func Level() slog.Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// GetOutput returns the writer used by the application log (stdout + app log
// file). It is intended for third-party loggers that must share the same
// destination. The returned writer is safe for concurrent use.
func GetOutput() io.Writer {
	mu.RLock()
	defer mu.RUnlock()
	if appOutput == nil {
		return os.Stdout
	}
	return appOutput
}

// SetDir overrides the directory log files are written to. It exists for tests;
// production code should rely on the default Dir.
func SetDir(path string) {
	mu.Lock()
	dir = path
	mu.Unlock()
}

// ParseLevel converts a textual level into a slog.Level.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case LevelDebugName:
		return slog.LevelDebug, nil
	case LevelInfoName:
		return slog.LevelInfo, nil
	case LevelWarnName, "warning":
		return slog.LevelWarn, nil
	case LevelErrorName:
		return slog.LevelError, nil
	default:
		return DefaultLevel, fmt.Errorf("unknown log level %q", s)
	}
}

// LevelName renders a slog.Level as its canonical lowercase name.
func LevelName(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return LevelDebugName
	case l < slog.LevelWarn:
		return LevelInfoName
	case l < slog.LevelError:
		return LevelWarnName
	default:
		return LevelErrorName
	}
}

// fanoutHandler is a slog.Handler that fans every record out to the combined
// application log and to the file dedicated to the record's level.
type fanoutHandler struct {
	combined slog.Handler
	perLevel map[slog.Level]slog.Handler
	minLevel slog.Level
}

// buildHandler opens (or creates) every log file and returns the fan-out
// handler, the writer backing the combined application log, and the file
// handles so Setup can close them on the next reconfiguration.
func buildHandler(logDir string, minLevel slog.Level, pretty bool) (*fanoutHandler, io.Writer, []*os.File, error) {
	// Pretty output is for local reading, so it deliberately has no ANSI colors:
	// escape codes would be baked into the log files and break greppability.
	newHandler := func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
		if pretty {
			return slog.NewTextHandler(w, opts)
		}
		return slog.NewJSONHandler(w, opts)
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, nil, nil, fmt.Errorf("create log directory %q: %w", logDir, err)
	}

	open := func(name string) (*os.File, error) {
		return os.OpenFile(filepath.Join(logDir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}

	appFile, err := open(FileApp)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open %s: %w", FileApp, err)
	}
	files := []*os.File{appFile}
	output := io.MultiWriter(os.Stdout, appFile)

	// Per-level handlers accept every level; fanoutHandler.Handle only forwards
	// records to the file matching the record's exact level.
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	h := &fanoutHandler{
		combined: newHandler(output, opts),
		perLevel: make(map[slog.Level]slog.Handler, len(orderedLevels)),
		minLevel: minLevel,
	}

	for _, lvl := range orderedLevels {
		name, ok := levelFiles[lvl]
		if !ok {
			continue
		}
		f, err := open(name)
		if err != nil {
			for _, opened := range files {
				_ = opened.Close()
			}
			return nil, nil, nil, fmt.Errorf("open %s: %w", name, err)
		}
		files = append(files, f)
		h.perLevel[lvl] = newHandler(f, opts)
	}

	return h, output, files, nil
}

// Enabled reports whether the handler is interested in the given level.
func (h *fanoutHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle writes the record to the combined log and to the matching per-level
// file.
func (h *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	if h.combined != nil {
		if err := h.combined.Handle(ctx, r.Clone()); err != nil {
			firstErr = err
		}
	}
	if lh, ok := h.perLevel[r.Level]; ok {
		if err := lh.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// WithAttrs returns a handler with the given attributes pre-formatted.
func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := &fanoutHandler{minLevel: h.minLevel, perLevel: make(map[slog.Level]slog.Handler, len(h.perLevel))}
	if h.combined != nil {
		next.combined = h.combined.WithAttrs(attrs)
	}
	for lvl, lh := range h.perLevel {
		next.perLevel[lvl] = lh.WithAttrs(attrs)
	}
	return next
}

// WithGroup returns a handler that qualifies attributes with the group name.
func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := &fanoutHandler{minLevel: h.minLevel, perLevel: make(map[slog.Level]slog.Handler, len(h.perLevel))}
	if h.combined != nil {
		next.combined = h.combined.WithGroup(name)
	}
	for lvl, lh := range h.perLevel {
		next.perLevel[lvl] = lh.WithGroup(name)
	}
	return next
}
