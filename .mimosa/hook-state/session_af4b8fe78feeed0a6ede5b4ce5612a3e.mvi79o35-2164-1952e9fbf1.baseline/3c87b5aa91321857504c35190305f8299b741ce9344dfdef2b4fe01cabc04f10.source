package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDCtxKey struct{}

// RequestID generates a unique request ID for each HTTP request.
// Uses crypto/rand — request IDs flow through logs and headers and the cost
// of one syscall per request is negligible compared to the disk I/O the
// request triggers; no reason to use a weaker source.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = generateID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithRequestID attaches a request ID to the context. Intended for tests and
// for callers that need to chain a fixed ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey{}, id)
}

// GetRequestID extracts the request ID from context. Returns "" if the
// RequestID middleware was not applied — callers should treat that as a
// programming error in tests but a soft fault in production.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDCtxKey{}).(string)
	return id
}

func generateID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand.Read essentially never fails on supported platforms;
		// returning a constant on error is preferable to crashing the
		// request handler.
		return "00000000"
	}
	return hex.EncodeToString(b[:])
}
