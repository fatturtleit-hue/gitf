package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gitf/internal/gitutil"
	"gitf/internal/models"
	"gitf/internal/repo"
	"gitf/internal/webhooks"
)

// gitinternalRepo creates a repo DB row + real bare repo with a two-commit
// main branch. Returns (repoPath, firstSHA, secondSHA).
func gitinternalRepo(t *testing.T, env *htEnv, owner, name string) (string, string, string) {
	t.Helper()
	if _, err := env.repos.Create(repo.OwnerTypeUser, env.user(t, owner).ID, name, "", "main", false); err != nil {
		t.Fatalf("seed repo row: %v", err)
	}
	repoPath := gitutil.RepoPath(env.cfg, owner, name)

	work := filepath.Join(t.TempDir(), "work")
	run := func(dir string, args ...string) string {
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@e"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	run(t.TempDir(), "clone", repoPath, work)
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("one\n"), 0o644)
	run(work, "add", "-A")
	run(work, "commit", "-m", "first")
	run(work, "push", "origin", "main")
	first := strings.TrimSpace(run(work, "rev-parse", "HEAD"))
	os.WriteFile(filepath.Join(work, "b.txt"), []byte("two\n"), 0o644)
	run(work, "add", "-A")
	run(work, "commit", "-m", "second")
	run(work, "push", "origin", "main")
	second := strings.TrimSpace(run(work, "rev-parse", "HEAD"))
	return repoPath, first, second
}

func preReceive(t *testing.T, env *htEnv, h *GitInternalHandler, owner, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/internal/pre-receive", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Gitf-Owner", owner)
	req.Header.Set("X-Gitf-Repo", name)
	w := httptest.NewRecorder()
	h.PreReceive(w, req)
	return w
}

func TestPreReceiveBranchProtection(t *testing.T) {
	env := newHandlerEnv(t)
	repoPath, first, second := gitinternalRepo(t, env, "alice", "prot")
	_ = repoPath

	// Protect main: no deletion, no force push.
	if err := env.db.Create(&models.BranchProtection{
		RepoID: 1, Pattern: "main", AllowDelete: false, AllowForcePush: false,
	}).Error; err != nil {
		t.Fatal(err)
	}

	h := NewGitInternalHandler(env.db, env.repos, env.cfg, nil)
	zero := "0000000000000000000000000000000000000000"

	// Deleting a protected branch is rejected.
	w := preReceive(t, env, h, "alice", "prot", first+" "+zero+" refs/heads/main")
	if w.Code != http.StatusForbidden {
		t.Errorf("protected delete: got %d, want 403", w.Code)
	}

	// A non-fast-forward update is rejected (unrelated commit).
	unrelated := strings.TrimSpace(gitHash(t, "unrelated"))
	w = preReceive(t, env, h, "alice", "prot", first+" "+unrelated+" refs/heads/main")
	if w.Code != http.StatusForbidden {
		t.Errorf("force push: got %d, want 403", w.Code)
	}

	// A normal fast-forward is accepted.
	w = preReceive(t, env, h, "alice", "prot", first+" "+second+" refs/heads/main")
	if w.Code != http.StatusOK {
		t.Errorf("fast-forward: got %d, want 200", w.Code)
	}
}

func gitHash(t *testing.T, seed string) string {
	dir := t.TempDir()
	cmd := exec.Command("git", "-c", "user.name=T", "-c", "user.email=t@e", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %s", out)
	}
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte(seed), 0o644)
	cmd = exec.Command("git", "-c", "user.name=T", "-c", "user.email=t@e", "add", "-A")
	cmd.Dir = dir
	_ = cmd.Run()
	cmd = exec.Command("git", "-c", "user.name=T", "-c", "user.email=t@e", "commit", "-m", seed)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %s", out)
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestPostReceiveEnqueuesDelivery(t *testing.T) {
	env := newHandlerEnv(t)
	repoPath, first, second := gitinternalRepo(t, env, "alice", "hooked")
	_ = repoPath

	env.db.Create(&models.Webhook{OwnerType: "repo", OwnerID: 1, URL: "http://127.0.0.1:9/h", Events: "push", Active: true})
	d := webhooks.NewDispatcher(env.db, 1, 5, 1)

	h := NewGitInternalHandler(env.db, env.repos, env.cfg, d)

	req := httptest.NewRequest(http.MethodPost, "/internal/post-receive",
		strings.NewReader(first+" "+second+" refs/heads/main\n"))
	req.Header.Set("X-Gitf-Owner", "alice")
	req.Header.Set("X-Gitf-Repo", "hooked")
	w := httptest.NewRecorder()
	h.PostReceive(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("post-receive: got %d", w.Code)
	}
	d.Shutdown()

	var deliveries []models.WebhookDelivery
	env.db.Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1 (push event enqueued)", len(deliveries))
	}
	if deliveries[0].Event != "push" || !strings.Contains(deliveries[0].Payload, "refs/heads/main") {
		t.Errorf("delivery = %+v", deliveries[0])
	}
}

func TestPostReceiveMissingHeaders(t *testing.T) {
	env := newHandlerEnv(t)
	h := NewGitInternalHandler(env.db, env.repos, env.cfg, nil)

	req := httptest.NewRequest(http.MethodPost, "/internal/post-receive", strings.NewReader("a b c\n"))
	w := httptest.NewRecorder()
	h.PostReceive(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing headers: got %d, want 400", w.Code)
	}
}
