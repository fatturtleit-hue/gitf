package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"gitf/internal/middleware"

	"gorm.io/gorm"
)

// HealthHandler provides health and readiness check endpoints.
type HealthHandler struct {
	DB *gorm.DB
	// LeakDetails controls whether readiness errors include the underlying
	// driver message. Defaults to false; only enable in development.
	LeakDetails bool
}

// NewHealthHandler creates a health check handler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	// Honor an explicit env opt-in. Default OFF — readiness probes are
	// reachable without authentication, and DB error strings can leak DSN
	// fragments or driver internals.
	return &HealthHandler{
		DB:          db,
		LeakDetails: os.Getenv("HEALTH_LEAK_DETAILS") == "true",
	}
}

// Liveness returns 200 as long as the process is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness checks if the server can serve traffic. Returns 200 when healthy
// and 503 when not. Always logs failures with full detail; the public response
// only includes the underlying error when HEALTH_LEAK_DETAILS=true is set.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := h.DB.DB()
	if err == nil {
		err = sqlDB.PingContext(r.Context())
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "readiness probe: db unreachable",
			"error", err,
			"request_id", middleware.GetRequestID(r.Context()),
		)
		body := map[string]string{"status": "degraded", "db": "error"}
		if h.LeakDetails {
			body["db_error"] = err.Error()
		}
		writeJSON(w, r, http.StatusServiceUnavailable, body)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
}
