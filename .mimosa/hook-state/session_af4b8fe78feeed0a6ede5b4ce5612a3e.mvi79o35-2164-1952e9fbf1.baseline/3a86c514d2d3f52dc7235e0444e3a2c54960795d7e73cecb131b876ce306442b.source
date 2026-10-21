// Package log provides structured logging via log/slog.
package log

import (
	"log/slog"
	"os"
)

// Init configures the global slog logger.
// level: "debug", "info", "warn", "error" (default "info").
// format: "json" or "text" (default "json").
func Init(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		// AddSource adds file:line; useful in development.
		AddSource: lvl == slog.LevelDebug,
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
