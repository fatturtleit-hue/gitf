package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/models"
)

type fakeUserStore struct {
	user *models.User
	err  error
}

func (f *fakeUserStore) ByUsername(ctx context.Context, username string) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.user != nil && f.user.Username == username {
		return f.user, nil
	}
	return nil, UserLookupError{username: username}
}

type UserLookupError struct{ username string }

func (e UserLookupError) Error() string { return "user not found: " + e.username }

func TestAuthMiddlewareJWT(t *testing.T) {
	secret := "test-secret-32-bytes-long!!!!"
	token, _ := auth.GenerateToken(1, "alice", "alice@example.com", secret)

	// JWT auth now looks the user up from the store so that role flags
	// (IsSiteAdmin, etc.) reflect the current DB state.
	store := &fakeUserStore{
		user: &models.User{ID: 1, Username: "alice", Email: "alice@example.com"},
	}

	handler := Auth(store,
		&config.Config{JWTSecret: secret},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.Username != "alice" {
			t.Errorf("username = %q, want alice", user.Username)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddlewareJWTUnknownUser(t *testing.T) {
	secret := "test-secret-32-bytes-long!!!!"
	token, _ := auth.GenerateToken(1, "ghost", "ghost@example.com", secret)

	// Store has no matching user — auth must fail.
	store := &fakeUserStore{
		user: &models.User{ID: 1, Username: "alice", Email: "alice@example.com"},
	}

	handler := Auth(store, &config.Config{JWTSecret: secret})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for unknown user")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown JWT user, got %d", w.Code)
	}
}

func TestAuthMiddlewareJWTAdminFlag(t *testing.T) {
	secret := "test-secret-32-bytes-long!!!!"
	token, _ := auth.GenerateToken(1, "root", "root@example.com", secret)

	store := &fakeUserStore{
		user: &models.User{ID: 1, Username: "root", Email: "root@example.com", IsSiteAdmin: true},
	}

	var gotIsAdmin bool
	handler := Auth(store, &config.Config{JWTSecret: secret})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		gotIsAdmin = user.IsSiteAdmin
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !gotIsAdmin {
		t.Error("expected IsSiteAdmin=true to be propagated from DB for JWT auth")
	}
}

func TestAuthMiddlewareMissing(t *testing.T) {
	handler := Auth(&fakeUserStore{},
		&config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without auth")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if h := w.Header().Get("WWW-Authenticate"); h == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestAuthMiddlewareBasic(t *testing.T) {
	hash, _ := auth.HashPassword("password123")
	store := &fakeUserStore{
		user: &models.User{ID: 1, Username: "bob", Email: "bob@example.com", PasswordHash: hash},
	}

	handler := Auth(store, &config.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		if user.Username != "bob" {
			t.Errorf("username = %q, want bob", user.Username)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/{owner}/{name}.git/info/refs", nil)
	req.SetBasicAuth("bob", "password123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddlewareBasicWrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("password123")
	store := &fakeUserStore{
		user: &models.User{ID: 1, Username: "bob", Email: "bob@example.com", PasswordHash: hash},
	}

	handler := Auth(store, &config.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/x/y.git/info/refs", nil)
	req.SetBasicAuth("bob", "wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOptionalAuthAllowsAnonymous(t *testing.T) {
	handler := OptionalAuth(
		&fakeUserStore{},
		&config.Config{JWTSecret: "test-secret-32-bytes-long!!!!"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); ok {
			t.Error("anonymous request should not have a user")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x/y.git/info/refs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy", "Permissions-Policy"} {
		if w.Header().Get(h) == "" {
			t.Errorf("expected %s header", h)
		}
	}
	csp := w.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") || strings.Contains(csp, "script-src 'unsafe-inline'") {
		t.Error("CSP should not allow unsafe-inline for scripts")
	}
	if !strings.Contains(csp, "script-src 'self'") {
		t.Error("CSP should restrict scripts to 'self'")
	}
}

func TestRequestID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRequestID(r.Context()) == "" {
			t.Error("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
}

func TestLogger(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected 418, got %d", w.Code)
	}
}

func TestStatusRecorderImplementsInterfaces(t *testing.T) {
	w := httptest.NewRecorder()
	r := NewStatusRecorder(w)
	if _, ok := interface{}(r).(http.Flusher); !ok {
		t.Error("statusRecorder should implement http.Flusher")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Avoid unused time import.
var _ = time.Second
