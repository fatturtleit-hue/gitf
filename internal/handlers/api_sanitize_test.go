package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSanitizeRefParam(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"main", "main", "main", true},
		{"nested branch", "feature/login-fix", "feature/login-fix", true},
		{"hash", "a1b2c3d4", "a1b2c3d4", true},
		{"tag dots", "v1.0.1", "v1.0.1", true},
		{"empty rejected", "", "", false},
		{"flag inject rejected", "--output=/tmp/x", "", false},
		{"dash rejected", "-x", "", false},
		{"traversal rejected", "main/../x", "", false},
		{"meta rejected", "a;b", "", false},
		{"space rejected", "a b", "", false},
		{"too long rejected", string(make([]byte, 300)), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sanitizeRefParam(tt.raw)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeRepoTreePath(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"empty root", "", "", true},
		{"simple", "README.md", "README.md", true},
		{"nested", "src/app/main.go", "src/app/main.go", true},
		{"spaces kept", "docs/my file.md", "docs/my file.md", true},
		{"traversal rejected", "../etc/passwd", "", false},
		{"inner traversal rejected", "src/../../x", "", false},
		{"dot segment rejected", "src/./x", "", false},
		{"double slash rejected", "src//x", "", false},
		{"absolute rejected", "/etc/passwd", "", false},
		{"flag rejected", "-x", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sanitizeRepoTreePath(tt.raw)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	env := newHandlerEnv(t)
	h := NewHealthHandler(env.db)

	w := httptest.NewRecorder()
	h.Liveness(w, newTestRequest(t, "GET", "/health", nil))
	if w.Code != http.StatusOK {
		t.Errorf("liveness: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.Readiness(w, newTestRequest(t, "GET", "/health/ready", nil))
	if w.Code != http.StatusOK {
		t.Errorf("readiness: got %d", w.Code)
	}
}

func TestAdminStats(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	env.seedRepo(t, "proj", false)
	h := NewAdminHandler(env.cfg, env.repos)

	req := chiContext(t, asUser(newTestRequest(t, http.MethodGet, "/admin/stats", nil), alice), nil)
	w := httptest.NewRecorder()
	h.GetStats(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("admin stats: got %d: %s", w.Code, w.Body.String())
	}
}
