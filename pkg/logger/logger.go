package logger

import (
	"log/slog"
	"os"
)

// New creates a structured JSON logger suitable for Cloud Run.
// Cloud Run captures stdout/stderr as structured logs when using JSON format.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
