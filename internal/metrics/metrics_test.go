package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMetricsMiddleware(t *testing.T) {
	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/repos", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestMetricsMiddlewareWithChiPattern(t *testing.T) {
	handler := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/bob", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/repos/{owner}/{name}"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") == "" {
		t.Error("expected content-type")
	}
}

func TestActiveGitOpsGauges(t *testing.T) {
	IncActiveGitOps()
	if got := gaugeActiveGitOps.Load(); got != 1 {
		t.Errorf("gauge = %d, want 1", got)
	}
	DecActiveGitOps()
	if got := gaugeActiveGitOps.Load(); got != 0 {
		t.Errorf("gauge = %d, want 0", got)
	}
}
