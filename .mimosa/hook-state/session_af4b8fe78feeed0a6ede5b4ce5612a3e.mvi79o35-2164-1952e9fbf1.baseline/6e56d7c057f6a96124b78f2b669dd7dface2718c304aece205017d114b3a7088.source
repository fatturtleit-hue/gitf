// Package metrics provides Prometheus-compatible metrics.
package metrics

import (
	"expvar"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"gitf/internal/middleware"

	"github.com/go-chi/chi/v5"
)

var (
	gaugeActiveGitOps atomic.Int64
	reqTotal          = expvar.NewMap("http_requests_total")
	reqDuration       = expvar.NewMap("http_request_duration_seconds")
)

// IncActiveGitOps increments the count of active git operations.
func IncActiveGitOps() { gaugeActiveGitOps.Add(1) }

// DecActiveGitOps decrements the count of active git operations.
func DecActiveGitOps() { gaugeActiveGitOps.Add(-1) }

// MetricsMiddleware records Prometheus-style metrics for each HTTP request.
// Reuses the shared status recorder from middleware/responsewriter.go so the
// status code observed here is the same value the logger sees.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewStatusRecorder(w)
		next.ServeHTTP(wrapped, r)

		// Use chi route pattern as metric label when available.
		pattern := chi.RouteContext(r.Context()).RoutePattern()
		if pattern == "" {
			pattern = r.URL.Path
		}

		code := strconv.Itoa(wrapped.Status())
		reqTotal.Add(pattern+"."+r.Method+"."+code, 1)
		reqDuration.Add(pattern+"."+r.Method, int64(time.Since(start)))
	})
}

// Handler serves metrics in expvar text format. Reachable at /metrics; do
// NOT expose this route to the public internet — it leaks per-route latency.
func Handler() http.Handler {
	return expvar.Handler()
}
