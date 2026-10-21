package gitutil

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"gitf/internal/config"
)

var (
	ErrRepoEmpty    = fmt.Errorf("repository is empty")
	ErrRefNotFound  = errors.New("ref not found")

	validNameRe   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,49}$`)
	reservedNames = map[string]bool{
		"repositories": true, "hooks": true, "objects": true,
		"refs": true, "info": true, "packed-refs": true,
		"HEAD": true, "config": true, "description": true,
	}
)

func ValidateName(s string) bool {
	if !validNameRe.MatchString(s) { return false }
	if strings.HasPrefix(s, ".") { return false }
	if strings.HasSuffix(s, ".git") { return false }
	return !reservedNames[s]
}

func RepoPath(cfg *config.Config, owner, name string) string {
	p := filepath.Join(cfg.ReposDir(), owner, name+".git")
	// Always absolute: git subprocesses run with `-C <repoPath>` (merge,
	// worktree), and a relative path used as a fetch remote would then resolve
	// against the bare repo's directory instead of the server CWD.
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func InitBare(cfg *config.Config, owner, name, defaultBranch string) error {
	path := RepoPath(cfg, owner, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create repo parent dir: %w", err)
	}
	cmd := exec.Command("git", "init", "--bare", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init --bare: %w: %s", err, string(out))
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	headRef := fmt.Sprintf("ref: refs/heads/%s\n", defaultBranch)
	if err := os.WriteFile(filepath.Join(path, "HEAD"), []byte(headRef), 0o644); err != nil {
		return fmt.Errorf("write HEAD: %w", err)
	}
	return InstallHooks(cfg, path, owner, name)
}

// InstallHooks (re)writes the server-side hooks for one repository. It is
// idempotent and also repairs repos created before hooks carried owner
// headers or before post-receive existed.
func InstallHooks(cfg *config.Config, repoPath, owner, name string) error {
	internalURL := internalURL(cfg)
	hooksDir := filepath.Join(repoPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}
	if cfg.HookSecret != "" {
		script := fmt.Sprintf(`#!/bin/sh
set -e
token='%s'
body=$(cat)
result=$(echo "$body" | curl -sSf -X POST \
  -H 'Content-Type: text/plain' \
  -H "X-Gitf-Hook-Token: $token" \
  -H "X-Gitf-Owner: %s" \
  -H "X-Gitf-Repo: %s" \
  --data-binary @- \
  %s/internal/pre-receive 2>&1) || {
  echo "Push rejected: $result" >&2
  exit 1
}
exit 0
`, cfg.HookSecret, owner, name, internalURL)
		if err := os.WriteFile(filepath.Join(hooksDir, "pre-receive"), []byte(script), 0o755); err != nil {
			return fmt.Errorf("install pre-receive hook: %w", err)
		}
	}

	// post-receive is best-effort: a failed webhook dispatch must never fail
	// the push itself.
	postScript := fmt.Sprintf(`#!/bin/sh
body=$(cat)
echo "$body" | curl -sS -X POST \
  -H 'Content-Type: text/plain' \
  %s \
  -H "X-Gitf-Owner: %s" \
  -H "X-Gitf-Repo: %s" \
  --data-binary @- \
  %s/internal/post-receive >/dev/null 2>&1 || true
exit 0
`, hookTokenLine(cfg.HookSecret), owner, name, internalURL)
	if err := os.WriteFile(filepath.Join(hooksDir, "post-receive"), []byte(postScript), 0o755); err != nil {
		return fmt.Errorf("install post-receive hook: %w", err)
	}
	return nil
}

func hookTokenLine(secret string) string {
	if secret == "" {
		return ""
	}
	return fmt.Sprintf(`-H "X-Gitf-Hook-Token: %s" \`, secret)
}

// internalURL normalizes the listen address into a loopback URL usable by
// server-side git hooks, which always run on the same host.
func internalURL(cfg *config.Config) string {
	addr := cfg.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return "http://" + addr
}

// SweepInstallHooks walks every bare repository on disk and (re)installs its
// server-side hooks. Run at startup so repos created by older versions get
// current hooks.
func SweepInstallHooks(cfg *config.Config) int {
	matches, err := filepath.Glob(filepath.Join(cfg.ReposDir(), "*", "*.git"))
	if err != nil {
		return 0
	}
	n := 0
	for _, repoPath := range matches {
		owner := filepath.Base(filepath.Dir(repoPath))
		name := strings.TrimSuffix(filepath.Base(repoPath), ".git")
		if !ValidateName(owner) || !ValidateName(name) {
			continue
		}
		if err := InstallHooks(cfg, repoPath, owner, name); err == nil {
			n++
		}
	}
	return n
}

func RemoveBare(cfg *config.Config, owner, name string) error {
	InvalidateRepoCache(owner, name)
	return os.RemoveAll(RepoPath(cfg, owner, name))
}

func Open(cfg *config.Config, owner, name string) (*git.Repository, error) {
	return getRepo(owner, name, RepoPath(cfg, owner, name))
}

// ── Types ───────────────────────────────────────────────────────────────────

type FileDiff struct {
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
	Status  string `json:"status"`
	Patch   string `json:"patch"`
}

type CommitInfo struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short_hash"`
	Message   string   `json:"message"`
	Author    string   `json:"author"`
	Email     string   `json:"email"`
	Date      string   `json:"date"`
	Parents   []string `json:"parents"`
}

type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
	Mode string `json:"mode"`
	Size int64  `json:"size"`
}

type TagInfo struct {
	Name        string `json:"name"`
	Hash        string `json:"hash"`
	Message     string `json:"message,omitempty"`
	Tagger      string `json:"tagger,omitempty"`
	Date        string `json:"date,omitempty"`
	IsAnnotated bool   `json:"is_annotated"`
}

type RepoStats struct {
	Size        int64 `json:"size"`
	FileCount   int64 `json:"file_count"`
	DirCount    int64 `json:"dir_count"`
	CommitCount int64 `json:"commit_count"`
	BranchCount int64 `json:"branch_count"`
}

// ── Git operations ──────────────────────────────────────────────────────────

func Stats(cfg *config.Config, owner, name, branch string) (RepoStats, error) {
	repo, err := Open(cfg, owner, name)
	if err != nil { return RepoStats{}, err }
	var stats RepoStats
	ref, err := resolveBranchRef(repo, branch)
	if err != nil { return RepoStats{}, nil }
	iter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		if errors.Is(err, object.ErrParentNotFound) || errors.Is(err, plumbing.ErrObjectNotFound) {
			return RepoStats{}, nil
		}
		return RepoStats{}, err
	}
	_ = iter.ForEach(func(c *object.Commit) error {
		stats.CommitCount++
		tree, err := c.Tree()
		if err != nil { return nil }
		for _, entry := range tree.Entries {
			if entry.Mode == filemode.Dir {
				stats.DirCount++
			} else {
				stats.FileCount++
				blob, err := repo.BlobObject(entry.Hash)
				if err == nil {
					stats.Size += blob.Size
				}
			}
		}
		return nil
	})
	branches, _ := Branches(cfg, owner, name)
	stats.BranchCount = int64(len(branches))
	return stats, nil
}

func Branches(cfg *config.Config, owner, name string) ([]string, error) {
	repo, err := Open(cfg, owner, name)
	if err != nil { return []string{}, err }
	result := []string{}
	iter, err := repo.Branches()
	if err != nil { return []string{}, err }
	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		result = append(result, ref.Name().Short())
		return nil
	})
	return result, nil
}

func Commits(cfg *config.Config, owner, name, branch string, limit int) ([]CommitInfo, error) {
	repo, err := Open(cfg, owner, name)
	if err != nil { return nil, err }
	ref, err := resolveBranchRef(repo, branch)
	if err != nil { return nil, ErrRepoEmpty }
	iter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil { return nil, err }
	var commits []CommitInfo
	count := 0
	_ = iter.ForEach(func(c *object.Commit) error {
		if limit > 0 && count >= limit { return storer.ErrStop }
		commits = append(commits, commitInfo(c))
		count++
		return nil
	})
	return commits, nil
}

func Commit(cfg *config.Config, owner, name, hash string) (*CommitInfo, error) {
	repo, err := Open(cfg, owner, name)
	if err != nil { return nil, err }
	h := plumbing.NewHash(hash)
	c, err := repo.CommitObject(h)
	if err != nil { return nil, err }
	info := commitInfo(c)
	return &info, nil
}

func TreeEntries(cfg *config.Config, owner, name, branch, path string) ([]TreeEntry, error) {
	if !ValidateRepoPathSegment(path) {
		return nil, fmt.Errorf("invalid path")
	}
	repo, err := Open(cfg, owner, name)
	if err != nil { return nil, err }
	ref, err := resolveBranchRef(repo, branch)
	if err != nil { return nil, ErrRepoEmpty }
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil { return nil, err }
	tree, err := commit.Tree()
	if err != nil { return nil, err }
	if path != "" {
		entry, err := tree.FindEntry(path)
		if err != nil { return nil, err }
		if entry.Mode != filemode.Dir { return nil, fmt.Errorf("not a directory: %s", path) }
		tree, err = repo.TreeObject(entry.Hash)
		if err != nil { return nil, err }
	}
	var entries []TreeEntry
	for _, entry := range tree.Entries {
		e := TreeEntry{
			Name: entry.Name,
			Path: filepath.Join(path, entry.Name),
			Mode: entry.Mode.String(),
		}
		if entry.Mode == filemode.Dir {
			e.Type = "tree"
		} else {
			e.Type = "blob"
			blob, err := repo.BlobObject(entry.Hash)
			if err == nil {
				e.Size = blob.Size
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func BlobContent(cfg *config.Config, owner, name, branch, path string) (string, error) {
	if !ValidateRepoPathSegment(path) {
		return "", fmt.Errorf("invalid path")
	}
	repo, err := Open(cfg, owner, name)
	if err != nil { return "", err }
	ref, err := resolveBranchRef(repo, branch)
	if err != nil { return "", err }
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil { return "", err }
	tree, err := commit.Tree()
	if err != nil { return "", err }
	entry, err := tree.FindEntry(path)
	if err != nil { return "", err }
	if entry.Mode == filemode.Dir { return "", fmt.Errorf("path is a directory: %s", path) }
	blob, err := repo.BlobObject(entry.Hash)
	if err != nil { return "", err }
	reader, err := blob.Reader()
	if err != nil { return "", err }
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil { return "", err }
	return string(data), nil
}

func BlobSize(cfg *config.Config, owner, name, branch, path string) (int64, error) {
	if !ValidateRepoPathSegment(path) {
		return 0, fmt.Errorf("invalid path")
	}
	repo, err := Open(cfg, owner, name)
	if err != nil { return 0, err }
	ref, err := resolveBranchRef(repo, branch)
	if err != nil { return 0, err }
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil { return 0, err }
	tree, err := commit.Tree()
	if err != nil { return 0, err }
	entry, err := tree.FindEntry(path)
	if err != nil { return 0, err }
	blob, err := repo.BlobObject(entry.Hash)
	if err != nil { return 0, err }
	return blob.Size, nil
}

func Tags(cfg *config.Config, owner, name string) ([]TagInfo, error) {
	repo, err := Open(cfg, owner, name)
	if err != nil { return []TagInfo{}, err }
	tags := []TagInfo{}
	iter, err := repo.Tags()
	if err != nil { return []TagInfo{}, err }
	_ = iter.ForEach(func(ref *plumbing.Reference) error {
		tag := TagInfo{Name: ref.Name().Short(), Hash: ref.Hash().String()}
		obj, err := repo.TagObject(ref.Hash())
		if err == nil {
			tag.Message = obj.Message
			tag.Tagger = obj.Tagger.Name
			tag.Date = obj.Tagger.When.Format(time.RFC3339)
			tag.IsAnnotated = true
		}
		tags = append(tags, tag)
		return nil
	})
	return tags, nil
}

func Diff(cfg *config.Config, owner, name, base, head string) ([]FileDiff, error) {
	if !ValidateRef(base) || !ValidateRef(head) {
		return nil, fmt.Errorf("invalid ref")
	}
	repoPath := RepoPath(cfg, owner, name)
	cmd := exec.Command("git", "-C", repoPath, "diff", "--unified=3", base+"..."+head)
	out, err := cmd.CombinedOutput()
	if err != nil { return nil, fmt.Errorf("git diff: %w: %s", err, string(out)) }
	if len(out) == 0 { return nil, nil }

	cmd2 := exec.Command("git", "-C", repoPath, "diff", "--name-status", base+"..."+head)
	out2, err := cmd2.CombinedOutput()
	if err != nil { return nil, err }

	patches := splitDiff(out)
	files := parseNameStatus(out2)
	for i := range patches {
		if i < len(files) {
			patches[i].Path = files[i].path
			patches[i].OldPath = files[i].oldPath
			patches[i].Status = files[i].status
		}
	}
	return patches, nil
}

func DiffStat(cfg *config.Config, owner, name, base, head string) (map[string]string, error) {
	if !ValidateRef(base) || !ValidateRef(head) {
		return nil, fmt.Errorf("invalid ref")
	}
	repoPath := RepoPath(cfg, owner, name)
	cmd := exec.Command("git", "-C", repoPath, "diff", "--shortstat", base+"..."+head)
	out, err := cmd.CombinedOutput()
	if err != nil { return nil, err }
	return map[string]string{"shortstat": strings.TrimSpace(string(out))}, nil
}

func Archive(cfg *config.Config, owner, name, ref, format string) (io.ReadSeekCloser, error) {
	if !ValidateRef(ref) {
		return nil, ErrRefNotFound
	}
	repo, err := Open(cfg, owner, name)
	if err != nil {
		return nil, err
	}
	tree, err := resolveTree(repo, ref)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "gitf-archive-*")
	if err != nil {
		return nil, err
	}
	archiveName := name
	if format == "zip" {
		archiveName += ".zip"
	} else {
		format = "tar"
		archiveName += ".tar.gz"
	}
	outPath := filepath.Join(tmpDir, archiveName)
	out, err := os.Create(outPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	writeErr := writeArchive(out, repo, tree, format, name+"/")
	closeErr := out.Close()
	if writeErr != nil || closeErr != nil {
		os.RemoveAll(tmpDir)
		if writeErr != nil {
			return nil, writeErr
		}
		return nil, closeErr
	}
	f, err := os.Open(outPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}
	return &archiveFile{File: f, dir: tmpDir}, nil
}

// resolveTree maps a branch name, tag name or commit hash to its tree.
func resolveTree(repo *git.Repository, ref string) (*object.Tree, error) {
	if r, err := repo.Reference(plumbing.NewBranchReferenceName(ref), true); err == nil {
		if c, err := repo.CommitObject(r.Hash()); err == nil {
			return c.Tree()
		}
	}
	if r, err := repo.Tag(ref); err == nil {
		if t, err := repo.TagObject(r.Hash()); err == nil {
			if c, err := t.Commit(); err == nil {
				return c.Tree()
			}
		}
	}
	if h := plumbing.NewHash(ref); !h.IsZero() {
		if c, err := repo.CommitObject(h); err == nil {
			return c.Tree()
		}
	}
	return nil, ErrRefNotFound
}

// writeArchive streams the repository tree into out as tar.gz or zip. It is
// implemented in-process with go-git so no request-derived string is ever
// passed to a git subprocess.
func writeArchive(out io.Writer, repo *git.Repository, tree *object.Tree, format, prefix string) error {
	if format == "zip" {
		zw := zip.NewWriter(out)
		defer zw.Close()
		return walkArchiveTree(repo, tree, func(name string, mode filemode.FileMode, size int64, content io.ReadCloser, link string) error {
			hdr := &zip.FileHeader{Name: prefix + name, Method: zip.Deflate}
			if mode == filemode.Symlink {
				hdr.SetMode(os.ModeSymlink | 0o644)
				w, err := zw.CreateHeader(hdr)
				if err != nil {
					return err
				}
				_, err = w.Write([]byte(link))
				return err
			}
			hdr.SetMode(fsArchiveMode(mode))
			w, err := zw.CreateHeader(hdr)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, content)
			return err
		})
	}

	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	return walkArchiveTree(repo, tree, func(name string, mode filemode.FileMode, size int64, content io.ReadCloser, link string) error {
		hdr := &tar.Header{
			Name:    prefix + name,
			Mode:    int64(fsArchiveMode(mode)),
			ModTime: time.Now(),
		}
		if mode == filemode.Symlink {
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = link
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			content.Close()
			return nil
		}
		hdr.Size = size
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err := io.Copy(tw, content)
		content.Close()
		return err
	})
}

func fsArchiveMode(mode filemode.FileMode) os.FileMode {
	if mode == filemode.Executable {
		return 0o755
	}
	return 0o644
}

// walkArchiveTree visits every blob in the tree depth-first. content is an
// open blob reader that the callback must consume and close.
func walkArchiveTree(repo *git.Repository, tree *object.Tree, visit func(name string, mode filemode.FileMode, size int64, content io.ReadCloser, link string) error) error {
	walker := object.NewTreeWalker(tree, true, nil)
	defer walker.Close()
	for {
		entryPath, entry, err := walker.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch entry.Mode {
		case filemode.Submodule:
			continue
		case filemode.Symlink:
			blob, err := repo.BlobObject(entry.Hash)
			if err != nil {
				continue
			}
			reader, err := blob.Reader()
			if err != nil {
				continue
			}
			target, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				continue
			}
			if err := visit(path.Clean(entryPath), entry.Mode, 0, nil, string(target)); err != nil {
				return err
			}
		case filemode.Regular, filemode.Executable:
			blob, err := repo.BlobObject(entry.Hash)
			if err != nil {
				continue
			}
			reader, err := blob.Reader()
			if err != nil {
				continue
			}
			if err := visit(path.Clean(entryPath), entry.Mode, blob.Size, reader, ""); err != nil {
				reader.Close()
				return err
			}
		}
	}
}

type archiveFile struct{ *os.File; dir string }
func (a *archiveFile) Close() error { a.File.Close(); return os.RemoveAll(a.dir) }
func (a *archiveFile) Seek(offset int64, whence int) (int64, error) { return a.File.Seek(offset, whence) }

// ── Helpers ─────────────────────────────────────────────────────────────────

// safeRefRe matches branch names, tag names and object hashes that may be
// passed as git arguments. Anything else (leading dashes would be parsed as
// flags — e.g. "archive --output=…" is a file-write primitive) is rejected.
var safeRefRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,200}$`)

// ValidateRef reports whether ref is safe to pass to a git subprocess.
// It must never start with "-" (flag injection) and must not contain ".."
// (path-ish traversal inside refspecs).
func ValidateRef(ref string) bool {
	if !safeRefRe.MatchString(ref) {
		return false
	}
	if strings.Contains(ref, "..") {
		return false
	}
	return true
}

// ValidateRepoPathSegment validates a path of a file INSIDE a repository tree
// (tree/blob APIs). Rejects traversal and flag-like segments; go-git resolves
// the rest in-process, so this is defense in depth.
func ValidateRepoPathSegment(path string) bool {
	if path == "" {
		return true
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "-") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return len(path) <= 1024
}

func resolveBranchRef(repo *git.Repository, branch string) (*plumbing.Reference, error) {
	for _, b := range []string{branch, "main", "master"} {
		ref, err := repo.Reference(plumbing.NewBranchReferenceName(b), true)
		if err == nil { return ref, nil }
	}
	return nil, plumbing.ErrReferenceNotFound
}

func commitInfo(c *object.Commit) CommitInfo {
	var parents []string
	_ = c.Parents().ForEach(func(p *object.Commit) error {
		parents = append(parents, p.Hash.String())
		return nil
	})
	if parents == nil { parents = []string{} }
	return CommitInfo{
		Hash:      c.Hash.String(),
		ShortHash: c.Hash.String()[:8],
		Message:   strings.SplitN(c.Message, "\n", 2)[0],
		Author:    c.Author.Name,
		Email:     c.Author.Email,
		Date:      c.Author.When.Format(time.RFC3339),
		Parents:   parents,
	}
}

func splitDiff(data []byte) []FileDiff {
	raw := strings.TrimSpace(string(data))
	if raw == "" { return nil }
	chunks := strings.Split("diff --git "+raw, "\ndiff --git ")
	var patches []FileDiff
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			patches = append(patches, FileDiff{Patch: "diff --git " + chunk})
		}
	}
	return patches
}

type nameStatus struct{ path, oldPath, status string }

func parseNameStatus(data []byte) []nameStatus {
	var result []nameStatus
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 { continue }
		status := parts[0]
		mapping := map[string]string{"A": "added", "M": "modified", "D": "deleted", "R": "renamed"}
		stStr, ok := mapping[status]
		if !ok { stStr = "modified" }
		ns := nameStatus{status: stStr}
		if status == "R" && len(parts) >= 3 {
			ns.oldPath = parts[1]
			ns.path = parts[2]
		} else {
			ns.path = parts[1]
		}
		result = append(result, ns)
	}
	return result
}
