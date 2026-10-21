package gitutil

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// headParents returns the parent count of the HEAD commit of the bare repo.
func headParents(t *testing.T, repoPath string) int {
	t.Helper()
	out := gitOut(t, repoPath, "rev-list", "--parents", "-n", "1", "HEAD")
	return len(strings.Fields(strings.TrimSpace(out))) - 1
}

func commitCount(t *testing.T, repoPath, ref string) int {
	t.Helper()
	out := gitOut(t, repoPath, "rev-list", "--count", ref)
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		t.Fatalf("rev-list --count %s: %v: %s", ref, err, out)
	}
	return n
}

func TestMergeStrategyMerge(t *testing.T) {
	cfg := newTestCfg(t)
	repoPath, _, _, _ := newTestRepoWithCommits(t, cfg)

	featureCount := commitCount(t, repoPath, "feature")
	sha, err := Merge(repoPath, repoPath, "feature", "main", "merge")
	if err != nil {
		t.Fatalf("Merge merge: %v", err)
	}
	if len(strings.TrimSpace(sha)) != 40 {
		t.Errorf("Merge returned non-full SHA: %q", sha)
	}
	if got := headParents(t, repoPath); got != 2 {
		t.Errorf("merge commit should have 2 parents, got %d", got)
	}
	if got := commitCount(t, repoPath, "main"); got != featureCount+1 {
		// After merging, main == feature history (which includes main's) plus
		// exactly one merge commit.
		t.Errorf("main commit count = %d, want %d (feature history + merge commit)", got, featureCount+1)
	}
	if _, err := exec.Command("git", "-C", repoPath, "show", "main:src/feature.txt").Output(); err != nil {
		t.Errorf("feature file missing on main after merge: %v", err)
	}
}

func TestMergeStrategySquash(t *testing.T) {
	cfg := newTestCfg(t)
	repoPath, _, _, _ := newTestRepoWithCommits(t, cfg)

	mainBefore := commitCount(t, repoPath, "main")
	if _, err := Merge(repoPath, repoPath, "feature", "main", "squash"); err != nil {
		t.Fatalf("Merge squash: %v", err)
	}
	if got := headParents(t, repoPath); got != 1 {
		t.Errorf("squash commit should have 1 parent, got %d", got)
	}
	if got := commitCount(t, repoPath, "main"); got != mainBefore+1 {
		t.Errorf("main commit count = %d, want %d", got, mainBefore+1)
	}
	if _, err := exec.Command("git", "-C", repoPath, "show", "main:src/feature.txt").Output(); err != nil {
		t.Errorf("feature file missing on main after squash: %v", err)
	}
}

func TestMergeStrategyRebase(t *testing.T) {
	cfg := newTestCfg(t)
	repoPath, _, _, _ := newTestRepoWithCommits(t, cfg)

	// main has 2 commits, feature adds 1 → after rebase main stays linear with 3.
	if _, err := Merge(repoPath, repoPath, "feature", "main", "rebase"); err != nil {
		t.Fatalf("Merge rebase: %v", err)
	}
	if got := commitCount(t, repoPath, "main"); got != 3 {
		t.Errorf("rebased main commit count = %d, want 3 (linear, no merge commit)", got)
	}
	if got := headParents(t, repoPath); got != 1 {
		t.Errorf("rebased commits should have 1 parent, got %d", got)
	}
	if _, err := exec.Command("git", "-C", repoPath, "show", "main:src/feature.txt").Output(); err != nil {
		t.Errorf("feature file missing on main after rebase: %v", err)
	}
}

func TestMergeInvalidBranch(t *testing.T) {
	cfg := newTestCfg(t)
	repoPath, _, _, _ := newTestRepoWithCommits(t, cfg)

	// Flag-like branch names must be rejected before touching git.
	if _, err := Merge(repoPath, repoPath, "-inject", "main", "merge"); err == nil {
		t.Error("Merge with flag-like head branch should fail")
	}
	if _, err := Merge(repoPath, repoPath, "feature", "..", "merge"); err == nil {
		t.Error("Merge with traversal base branch should fail")
	}
}
