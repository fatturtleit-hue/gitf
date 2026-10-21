package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// statusRecorder wraps an http.ResponseWriter to capture the status code and
// the number of bytes written. Implements http.Flusher and http.Hijacker so
// SSE / websocket-upgrade middleware downstream still work.
//
// One implementation, used by both Logger and metrics.MetricsMiddleware so
// status capture is identical.
type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	bytes       int
	wroteHeader bool
}

// NewStatusRecorder returns a recording wrapper around w.
func NewStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

// Status returns the captured status code. Defaults to 200 if WriteHeader was
// never called (Go's net/http auto-promotes the first Write).
func (r *statusRecorder) Status() int { return r.statusCode }

// BytesWritten returns the total number of bytes written to the body.
func (r *statusRecorder) BytesWritten() int { return r.bytes }

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.statusCode = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.wroteHeader = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Flush implements http.Flusher when the underlying writer supports it.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker when the underlying writer supports it.
// Required for websocket upgrades and chi's StripSlashes + recoverer chain.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}
