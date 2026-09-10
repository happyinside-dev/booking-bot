// Package logger configures the application's structured logger (slog).
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a slog.Logger. In "production" it emits JSON (for log
// aggregation); otherwise it emits human-readable text. The level is parsed
// from a string ("debug", "info", "warn", "error"), defaulting to info on an
// unrecognized value.
func New(env, level string) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	var handler slog.Handler
	if strings.EqualFold(env, "production") {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
