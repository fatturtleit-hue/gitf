package gitutil

import (
	"os"
	"path/filepath"
	"testing"

	"gitf/internal/config"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"hello", true},
		{"hello-world", true},
		{"hello_world", true},
		{"Hello123", true},
		{"test-repo_01", true},
		{"", false},
		{"../evil", false},
		{"hello world", false},
		{"hello@world", false},
		{"hello/world", false},
		{".hidden", false},
		{"repo.git", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateName(tt.name); got != tt.valid {
				t.Errorf("ValidateName(%q)=%v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestRepoPath(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("DATA_DIR", dir)
	os.Unsetenv("DB_DSN")
	defer func() { os.Unsetenv("DATA_DIR") }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	got := RepoPath(cfg, "alice", "hello")
	expected := filepath.Join(dir, "repositories", "alice", "hello.git")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestInitAndRemoveBare(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DATA_DIR", tmpDir)
	os.Unsetenv("DB_DSN")
	defer func() { os.Unsetenv("DATA_DIR") }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	err = InitBare(cfg, "testuser", "testrepo", "main")
	if err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	repoPath := RepoPath(cfg, "testuser", "testrepo")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Fatal("bare repo was not created")
	}

	head, err := os.ReadFile(filepath.Join(repoPath, "HEAD"))
	if err != nil {
		t.Fatal("failed to read HEAD")
	}
	if string(head) != "ref: refs/heads/main\n" {
		t.Errorf("expected HEAD=ref: refs/heads/main, got %s", string(head))
	}

	err = RemoveBare(cfg, "testuser", "testrepo")
	if err != nil {
		t.Fatalf("RemoveBare failed: %v", err)
	}
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Fatal("repo was not removed")
	}
}

func TestInitBareDefaultBranch(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DATA_DIR", tmpDir)
	os.Unsetenv("DB_DSN")
	defer func() { os.Unsetenv("DATA_DIR") }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	err = InitBare(cfg, "user", "repo", "develop")
	if err != nil {
		t.Fatalf("InitBare failed: %v", err)
	}

	head, _ := os.ReadFile(filepath.Join(RepoPath(cfg, "user", "repo"), "HEAD"))
	expected := "ref: refs/heads/develop\n"
	if string(head) != expected {
		t.Errorf("expected %q, got %q", expected, string(head))
	}
}
