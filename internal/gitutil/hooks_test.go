package gitutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallHooksWithoutSecret(t *testing.T) {
	cfg := newTestCfg(t) // HOOK_SECRET unset
	repoPath := filepath.Join(t.TempDir(), "repo.git")
	if err := os.MkdirAll(filepath.Join(repoPath, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := InstallHooks(cfg, repoPath, "alice", "hello"); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	// Without a secret there is no pre-receive hook.
	if _, err := os.Stat(filepath.Join(repoPath, "hooks", "pre-receive")); !os.IsNotExist(err) {
		t.Error("pre-receive hook should not exist without HOOK_SECRET")
	}

	post, err := os.ReadFile(filepath.Join(repoPath, "hooks", "post-receive"))
	if err != nil {
		t.Fatalf("post-receive hook missing: %v", err)
	}
	for _, want := range []string{"X-Gitf-Owner: alice", "X-Gitf-Repo: hello", "internal/post-receive", "exit 0"} {
		if !strings.Contains(string(post), want) {
			t.Errorf("post-receive hook missing %q", want)
		}
	}
}

func TestInstallHooksWithSecret(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.HookSecret = "topsecret"
	repoPath := filepath.Join(t.TempDir(), "repo.git")
	if err := os.MkdirAll(filepath.Join(repoPath, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := InstallHooks(cfg, repoPath, "alice", "hello"); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	pre, err := os.ReadFile(filepath.Join(repoPath, "hooks", "pre-receive"))
	if err != nil {
		t.Fatal("pre-receive hook missing with HOOK_SECRET set")
	}
	for _, want := range []string{"token='topsecret'", `X-Gitf-Hook-Token: $token`, "X-Gitf-Owner: alice", "X-Gitf-Repo: hello", "internal/pre-receive"} {
		if !strings.Contains(string(pre), want) {
			t.Errorf("pre-receive hook missing %q", want)
		}
	}

	post, err := os.ReadFile(filepath.Join(repoPath, "hooks", "post-receive"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(post), "X-Gitf-Hook-Token: topsecret") {
		t.Error("post-receive hook should carry the secret token header")
	}

	info, _ := os.Stat(filepath.Join(repoPath, "hooks", "pre-receive"))
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		// Windows (NTFS) does not carry Unix permission bits — os.Stat always
		// reports 0666 for files without an executable extension there.
		t.Errorf("hook mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestSweepInstallHooks(t *testing.T) {
	cfg := newTestCfg(t)
	cfg.HookSecret = "sweepsecret"

	// Two valid repos via InitBare, one invalid owner directory.
	if err := InitBare(cfg, "alice", "r1", "main"); err != nil {
		t.Fatal(err)
	}
	if err := InitBare(cfg, "alice", "r2", "main"); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(cfg.ReposDir(), "..", "owner", "x.git")
	_ = invalid // outside ReposDir — the sweep glob must not reach it

	// Damage the hooks on r1 (delete the whole dir): the sweep recreates it.
	if err := os.RemoveAll(filepath.Join(RepoPath(cfg, "alice", "r1"), "hooks")); err != nil {
		t.Fatal(err)
	}

	n := SweepInstallHooks(cfg)
	if n != 2 {
		t.Errorf("SweepInstallHooks repaired %d repos, want 2", n)
	}
	if _, err := os.Stat(filepath.Join(RepoPath(cfg, "alice", "r1"), "hooks", "post-receive")); err != nil {
		t.Error("r1 hooks were not restored")
	}
}

func TestSweepIgnoresInvalidNames(t *testing.T) {
	cfg := newTestCfg(t)

	// An invalid owner directory must be skipped, not crash the sweep.
	invalid := filepath.Join(cfg.ReposDir(), "Bad Owner", "x.git")
	if err := os.MkdirAll(filepath.Join(invalid, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if n := SweepInstallHooks(cfg); n != 0 {
		t.Errorf("sweep should skip invalid names, got %d", n)
	}
}
