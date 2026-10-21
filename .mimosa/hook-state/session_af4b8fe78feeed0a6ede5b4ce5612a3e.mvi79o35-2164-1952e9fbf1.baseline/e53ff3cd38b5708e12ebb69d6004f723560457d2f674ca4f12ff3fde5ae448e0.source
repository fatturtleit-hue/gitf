package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitf/internal/middleware"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest(t)
	writeJSON(w, r, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	expected := `{"key":"value"}` + "\n"
	if w.Body.String() != expected {
		t.Errorf("expected body %q, got %q", expected, w.Body.String())
	}
}

func TestWriteAPIError(t *testing.T) {
	w := httptest.NewRecorder()
	r := newRequest(t)
	writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	expected := `{"code":"NOT_FOUND","message":"repository not found"}` + "\n"
	if w.Body.String() != expected {
		t.Errorf("expected body %q, got %q", expected, w.Body.String())
	}
}

func TestWriteAPIErrorAllCodes(t *testing.T) {
	tests := []struct {
		status int
		code   string
	}{
		{http.StatusBadRequest, ErrCodeValidation},
		{http.StatusUnauthorized, ErrCodeUnauthorized},
		{http.StatusForbidden, ErrCodeForbidden},
		{http.StatusNotFound, ErrCodeNotFound},
		{http.StatusConflict, ErrCodeConflict},
		{http.StatusTooManyRequests, ErrCodeRateLimited},
		{http.StatusInternalServerError, ErrCodeInternal},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		r := newRequest(t)
		writeAPIError(w, r, tt.status, tt.code, "test message")
		body := w.Body.String()
		expected := `"code":"` + tt.code + `"`
		if !strings.Contains(body, expected) {
			t.Errorf("status %d: expected body to contain %s, got %s", tt.status, expected, body)
		}
	}
}

func TestNewPagination(t *testing.T) {
	p := NewPagination(1, 50, 142)
	if p.Page != 1 || p.PerPage != 50 || p.Total != 142 || p.TotalPages != 3 {
		t.Errorf("unexpected pagination: %+v", p)
	}
	if !p.HasNext || p.HasPrev {
		t.Error("expected HasNext=true, HasPrev=false")
	}

	p2 := NewPagination(3, 50, 142)
	if p2.HasNext {
		t.Error("expected HasNext=false on last page")
	}
	if !p2.HasPrev {
		t.Error("expected HasPrev=true on page 3")
	}

	p3 := NewPagination(0, 0, 10)
	if p3.Page != 1 || p3.PerPage != 50 {
		t.Errorf("expected clamped values, got page=%d perPage=%d", p3.Page, p3.PerPage)
	}
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	return r.WithContext(middleware.WithRequestID(r.Context(), "test-req-1"))
}
