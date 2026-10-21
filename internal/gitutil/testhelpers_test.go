package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gitf/internal/config"
)

// newTestCfg points DATA_DIR at a per-test temp directory and makes sure no
// hook secret leaks in from the environment (a set HOOK_SECRET installs
// pre-receive hooks that reject every local push in tests).
func newTestCfg(t *testing.T) *config.Config {
	t.Helper()
	os.Setenv("DATA_DIR", t.TempDir())
	os.Setenv("JWT_SECRET", "test-secret-for-gitutil")
	os.Unsetenv("DB_DSN")
	os.Unsetenv("HOOK_SECRET")
	t.Cleanup(func() {
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("HOOK_SECRET")
	})
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

var repoSeq int

// uniqueName returns a fresh owner/name pair per call so the package-global
// repo cache never serves a stale handle across tests.
func uniqueName(base string) (string, string) {
	repoSeq++
	return fmt.Sprintf("towner%d", repoSeq), fmt.Sprintf("%s%d", base, repoSeq)
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-c", "user.name=Tester", "-c", "user.email=tester@example.com"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

// commitFile writes a file into the working tree and commits it.
func commitFile(t *testing.T, workDir, file, content, msg string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(workDir, file)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, workDir, "add", "-A")
	gitOut(t, workDir, "commit", "-m", msg)
}

// newTestRepoWithCommits creates a bare repo via InitBare and returns its
// path together with a clone that has:
//   - main: README.md (init) + docs/second.md (second commit)
//   - feature: everything from main plus src/feature.txt, annotated tag vtest.1
//
// The bare repo is removed on test cleanup.
func newTestRepoWithCommits(t *testing.T, cfg *config.Config) (repoPath, workDir, owner, name string) {
	t.Helper()
	owner, name = uniqueName("repo")
	if err := InitBare(cfg, owner, name, "main"); err != nil {
		t.Fatalf("InitBare: %v", err)
	}
	t.Cleanup(func() {
		_ = RemoveBare(cfg, owner, name)
	})
	repoPath = RepoPath(cfg, owner, name)

	workDir = filepath.Join(t.TempDir(), "work")
	gitOut(t, t.TempDir(), "clone", repoPath, workDir)

	commitFile(t, workDir, "README.md", "hello gitf test\n", "init")
	commitFile(t, workDir, "docs/second.md", "second commit\n", "second")
	gitOut(t, workDir, "push", "origin", "main")

	gitOut(t, workDir, "checkout", "-b", "feature")
	commitFile(t, workDir, "src/feature.txt", "feature content\n", "feature work")
	gitOut(t, workDir, "push", "-u", "origin", "feature")

	gitOut(t, workDir, "tag", "-a", "vtest.1", "-m", "test tag")
	gitOut(t, workDir, "push", "origin", "vtest.1")

	gitOut(t, workDir, "checkout", "main")
	return repoPath, workDir, owner, name
}
