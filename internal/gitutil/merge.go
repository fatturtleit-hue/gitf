package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// mergeIdentity keeps merge commits attributable — the server process has no
// global git identity configured.
var mergeIdentity = []string{
	"-c", "user.name=gitf",
	"-c", "user.email=gitf@localhost",
}

// runIn runs git <args...> inside dir and returns combined output.
func runIn(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// Merge a pull request into the base branch using the given strategy.
// baseRepoPath is the bare repository path; headRepoPath is the fork/source.
// Returns the merge commit SHA.
func Merge(baseRepoPath, headRepoPath, headBranch, baseBranch, strategy string) (string, error) {
	// Branch names become git argv values — reject anything flag-like.
	if !ValidateRef(headBranch) || !ValidateRef(baseBranch) {
		return "", fmt.Errorf("invalid branch name")
	}
	// Create a temporary worktree to perform the merge without touching bare HEAD.
	tmpDir, err := os.MkdirTemp("", "gitf-merge-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	worktreePath := filepath.Join(tmpDir, "worktree")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		return "", fmt.Errorf("create worktree dir: %w", err)
	}

	// Register the head repo as a remote on the bare repo (config is shared
	// with linked worktrees).
	if _, err := runIn(baseRepoPath, "remote", "add", "head", headRepoPath); err != nil {
		return "", fmt.Errorf("add remote: %s", err)
	}
	defer runIn(baseRepoPath, "remote", "remove", "head")
	defer runIn(baseRepoPath, "worktree", "remove", "--force", worktreePath)

	// Create the worktree BEFORE fetching: a linked worktree keeps FETCH_HEAD
	// in its own private git dir, so the fetch must happen inside it.
	if _, err := runIn(baseRepoPath, "worktree", "add", worktreePath, baseBranch); err != nil {
		return "", fmt.Errorf("add worktree: %s", err)
	}

	if out, err := runIn(worktreePath, "fetch", "head", headBranch); err != nil {
		return "", fmt.Errorf("fetch head: %w: %s", err, out)
	}

	// Perform the merge based on strategy.
	var out []byte
	switch strategy {
	case "squash":
		if _, err := runIn(worktreePath, "merge", "--squash", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("merge --squash: %s", err)
		}
		if out, err = runIn(worktreePath, append(mergeIdentity, "commit", "-m",
			fmt.Sprintf("Merge pull request (%s) via squash", headBranch))...); err != nil {
			return "", fmt.Errorf("commit squash: %w: %s", err, out)
		}
	case "rebase":
		// Check out the head tip, then replay its own commits on top of base.
		// The rebase run needs an identity too — it re-creates commits.
		if _, err := runIn(worktreePath, "checkout", "-B", "gitf-pr-head", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("checkout head: %s", err)
		}
		if out, err = runIn(worktreePath, append(mergeIdentity, "rebase", baseBranch)...); err != nil {
			return "", fmt.Errorf("rebase: %w: %s", err, out)
		}
	default: // "merge"
		if out, err = runIn(worktreePath, append(mergeIdentity, "merge", "--no-ff", "-m",
			fmt.Sprintf("Merge pull request (%s)", headBranch), "FETCH_HEAD")...); err != nil {
			return "", fmt.Errorf("merge: %w: %s", err, out)
		}
	}

	// Push back to the bare repo directly — a bare repo has no "origin".
	if out, err := runIn(worktreePath, "push", baseRepoPath, "HEAD:"+baseBranch); err != nil {
		return "", fmt.Errorf("push merge: %w: %s", err, out)
	}

	// Get the resulting commit SHA.
	out, err = runIn(worktreePath, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("rev-parse: %w: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
