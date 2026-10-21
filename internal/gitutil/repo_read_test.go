package gitutil

import (
	"path/filepath"
	"strings"
	"testing"

	"gitf/internal/config"
)

func TestRepoPathIsAbsolute(t *testing.T) {
	// The whole point of the absolutization: even with a relative DATA_DIR the
	// repo path must be absolute, because it is used as cmd.Dir and as a
	// fetch remote from other working directories.
	t.Setenv("DATA_DIR", "relative-data-dir")
	t.Setenv("DB_DSN", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	p := RepoPath(cfg, "alice", "hello")
	if !filepath.IsAbs(p) {
		t.Errorf("RepoPath should return an absolute path, got %q", p)
	}
}

func TestOpenAndCache(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	r1, err := Open(cfg, owner, name)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	r2, err := Open(cfg, owner, name)
	if err != nil {
		t.Fatalf("Open (cached): %v", err)
	}
	if r1 != r2 {
		t.Error("second Open should return the cached repository handle")
	}

	InvalidateRepoCache(owner, name)
	r3, err := Open(cfg, owner, name)
	if err != nil {
		t.Fatalf("Open after invalidate: %v", err)
	}
	if r3 == r1 {
		t.Error("after InvalidateRepoCache a fresh handle should be created")
	}
}

func TestBranchesAndCommits(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	branches, err := Branches(cfg, owner, name)
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	joined := strings.Join(branches, ",")
	if !strings.Contains(joined, "main") || !strings.Contains(joined, "feature") {
		t.Errorf("branches = %v, want main+feature", branches)
	}

	commits, err := Commits(cfg, owner, name, "main", 0)
	if err != nil {
		t.Fatalf("Commits: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("main commits = %d, want 2", len(commits))
	}

	limited, err := Commits(cfg, owner, name, "main", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 {
		t.Errorf("limit=1 should return 1 commit, got %d", len(limited))
	}
}

func TestCommitByHash(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	commits, err := Commits(cfg, owner, name, "main", 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Commit(cfg, owner, name, commits[0].Hash)
	if err != nil {
		t.Fatalf("Commit by hash: %v", err)
	}
	if c.Hash != commits[0].Hash {
		t.Errorf("hash mismatch: %s vs %s", c.Hash, commits[0].Hash)
	}
}

func TestTreeAndBlob(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	entries, err := TreeEntries(cfg, owner, name, "main", "")
	if err != nil {
		t.Fatalf("TreeEntries root: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Name == "README.md" && e.Type == "blob" {
			found = true
		}
	}
	if !found {
		t.Errorf("README.md not in root tree: %+v", entries)
	}

	sub, err := TreeEntries(cfg, owner, name, "feature", "src")
	if err != nil {
		t.Fatalf("TreeEntries src: %v", err)
	}
	if len(sub) != 1 || sub[0].Name != "feature.txt" {
		t.Errorf("src tree = %+v", sub)
	}

	content, err := BlobContent(cfg, owner, name, "feature", "src/feature.txt")
	if err != nil {
		t.Fatalf("BlobContent: %v", err)
	}
	if content != "feature content\n" {
		t.Errorf("blob content = %q", content)
	}

	size, err := BlobSize(cfg, owner, name, "feature", "src/feature.txt")
	if err != nil {
		t.Fatal(err)
	}
	if int(size) != len("feature content\n") {
		t.Errorf("blob size = %d", size)
	}
}

func TestTags(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	tags, err := Tags(cfg, owner, name)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	found := false
	for _, tag := range tags {
		if tag.Name == "vtest.1" {
			found = true
			if !tag.IsAnnotated || !strings.Contains(tag.Message, "test tag") {
				t.Errorf("annotated tag info incomplete: %+v", tag)
			}
		}
	}
	if !found {
		t.Errorf("vtest.1 not found in tags: %+v", tags)
	}
}

func TestStats(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	stats, err := Stats(cfg, owner, name, "main")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.CommitCount < 1 {
		t.Errorf("CommitCount = %d, want >= 1", stats.CommitCount)
	}
	if stats.BranchCount < 1 {
		t.Errorf("BranchCount = %d, want >= 1", stats.BranchCount)
	}
}

func TestDiffAndDiffStat(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	diffs, err := Diff(cfg, owner, name, "main", "feature")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	found := false
	for _, d := range diffs {
		if strings.Contains(d.Path, "feature.txt") {
			found = true
		}
	}
	if !found {
		t.Errorf("feature.txt not in diff: %+v", diffs)
	}

	stat, err := DiffStat(cfg, owner, name, "main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if stat["shortstat"] == "" {
		t.Error("shortstat should not be empty")
	}

	// Invalid refs are rejected before touching git.
	if _, err := Diff(cfg, owner, name, "-x", "feature"); err == nil {
		t.Error("Diff with flag-like base should fail")
	}
}
