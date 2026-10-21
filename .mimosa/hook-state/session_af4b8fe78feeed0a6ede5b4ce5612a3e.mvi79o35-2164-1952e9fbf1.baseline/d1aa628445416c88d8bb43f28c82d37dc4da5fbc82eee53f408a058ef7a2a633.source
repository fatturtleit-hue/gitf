package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger logs incoming HTTP requests using structured logging.
// Every log line carries the request_id so application-level log entries can
// be correlated with the inbound request line.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := NewStatusRecorder(w)
		next.ServeHTTP(wrapped, r)

		level := slog.LevelInfo
		if wrapped.Status() >= 500 {
			level = slog.LevelError
		} else if wrapped.Status() >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.Status()),
			slog.Int("bytes", wrapped.BytesWritten()),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("request_id", GetRequestID(r.Context())),
		)
	})
}
