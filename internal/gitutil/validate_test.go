package gitutil

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing/filemode"
)

func TestValidateRef(t *testing.T) {
	tests := []struct {
		name  string
		ref   string
		valid bool
	}{
		{"main", "main", true},
		{"feature/x", "feature/x", true},
		{"tag with dots", "v1.0.1", true},
		{"hash", "a1b2c3d4e5", true},
		{"underscore dash", "my_branch-2", true},
		{"empty", "", false},
		{"flag inject", "--output=/tmp/x", false},
		{"single dash", "-o", false},
		{"traversal", "main/../other", false},
		{"dotdot only", "..", false},
		{"space", "main main", false},
		{"shell meta", "main;rm", false},
		{"too long", string(make([]byte, 300)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateRef(tt.ref); got != tt.valid {
				t.Errorf("ValidateRef(%q)=%v, want %v", tt.ref, got, tt.valid)
			}
		})
	}
}

func TestValidateRepoPathSegment(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"empty means root", "", true},
		{"simple file", "README.md", true},
		{"nested", "src/app/main.go", true},
		{"spaces ok", "docs/my file.md", true},
		{"traversal", "../etc/passwd", false},
		{"inner traversal", "src/../../x", false},
		{"dot segment", "src/./x", false},
		{"empty segment", "src//x", false},
		{"absolute", "/etc/passwd", false},
		{"flag-like", "-weird", false},
		{"too long", string(make([]byte, 2000)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateRepoPathSegment(tt.path); got != tt.valid {
				t.Errorf("ValidateRepoPathSegment(%q)=%v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestHookTokenLine(t *testing.T) {
	if got := hookTokenLine(""); got != "" {
		t.Errorf("empty secret should produce no header line, got %q", got)
	}
	got := hookTokenLine("s3cret")
	want := `-H "X-Gitf-Hook-Token: s3cret" \`
	if got != want {
		t.Errorf("hookTokenLine = %q, want %q", got, want)
	}
}

func TestInternalURL(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.Addr = ":8080"
	if got := internalURL(cfg); got != "http://127.0.0.1:8080" {
		t.Errorf("internalURL(:8080) = %q", got)
	}
	cfg.Addr = "0.0.0.0:9000"
	if got := internalURL(cfg); got != "http://0.0.0.0:9000" {
		t.Errorf("internalURL(0.0.0.0:9000) = %q", got)
	}
}

func TestFsArchiveMode(t *testing.T) {
	if got := fsArchiveMode(filemode.Executable); got != 0o755 {
		t.Errorf("executable mode = %v", got)
	}
	if got := fsArchiveMode(filemode.Regular); got != 0o644 {
		t.Errorf("regular mode = %v", got)
	}
}
