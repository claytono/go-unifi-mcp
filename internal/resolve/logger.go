package resolve

import (
	"log/slog"
	"os"
)

// NewLogger creates an slog.Logger that respects the given log level string.
// The level string should be one of: disabled, trace, debug, info, warn, error.
func NewLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "disabled":
		// Return a logger with a level higher than any message
		level = slog.Level(100)
	case "trace", "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	default: // "error" or unknown
		level = slog.LevelError
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
