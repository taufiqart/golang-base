package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
)

var globalOutput io.Writer

// Setup initializes the global slog logger to write to both stdout and a file.
func Setup() {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("failed to create logs directory: %v", err)
	}

	// Open or create the log file
	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	// Write to both stdout and the file
	globalOutput = io.MultiWriter(os.Stdout, file)

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(globalOutput, opts)
	slog.SetDefault(slog.New(handler))
}

// GetOutput returns the writer used by the logger (Stdout + File).
func GetOutput() io.Writer {
	if globalOutput == nil {
		return os.Stdout
	}
	return globalOutput
}
