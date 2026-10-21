package gitutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countArchiveTempDirs counts leftover gitf-archive-* dirs in the OS temp dir
// so tests can assert cleanup happened.
func countArchiveTempDirs(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "gitf-archive-*"))
	if err != nil {
		t.Fatal(err)
	}
	return len(matches)
}

func TestArchiveTarGz(t *testing.T) {
	cfg := newTestCfg(t)
	repoPath, _, owner, name := newTestRepoWithCommits(t, cfg)
	_ = repoPath

	before := countArchiveTempDirs(t)
	reader, err := Archive(cfg, owner, name, "main", "tar.gz")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if after := countArchiveTempDirs(t); after != before {
		t.Errorf("archive temp dir leaked: before=%d after=%d", before, after)
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not gzip: %v", err)
	}
	tr := tar.NewReader(gz)
	seen := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		seen[hdr.Name] = string(body)
	}
	if content, ok := seen[name+"/README.md"]; !ok || !strings.Contains(content, "hello gitf test") {
		t.Errorf("README.md missing or wrong in archive: %q", seen[name+"/README.md"])
	}
	// main does not contain the feature branch file
	if _, ok := seen[name+"/src/feature.txt"]; ok {
		t.Error("feature file should not be in a main archive")
	}
}

func TestArchiveZip(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	reader, err := Archive(cfg, owner, name, "feature", "zip")
	if err != nil {
		t.Fatalf("Archive zip: %v", err)
	}
	// zip.Reader needs random access — use Seek to determine size.
	size, err := reader.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), size)
	if err != nil {
		t.Fatalf("not zip: %v", err)
	}
	seen := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		seen[f.Name] = string(body)
	}
	for _, want := range []string{name + "/README.md", name + "/src/feature.txt"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("zip archive missing %s; got %v", want, keys(seen))
		}
	}
	if content := seen[name+"/src/feature.txt"]; content != "feature content\n" {
		t.Errorf("feature.txt content = %q", content)
	}
}

func TestArchiveRefValidation(t *testing.T) {
	cfg := newTestCfg(t)
	_, _, owner, name := newTestRepoWithCommits(t, cfg)

	// Flag-injection and traversal refs are rejected by the allowlist.
	for _, bad := range []string{"--output=/tmp/pwned", "main/../x", "-o"} {
		if _, err := Archive(cfg, owner, name, bad, "tar.gz"); err == nil {
			t.Errorf("Archive with ref %q should fail", bad)
		}
	}
	// Well-formed but nonexistent ref.
	if _, err := Archive(cfg, owner, name, "does-not-exist", "tar.gz"); err == nil {
		t.Error("Archive with missing ref should fail")
	} else if err != ErrRefNotFound {
		t.Errorf("want ErrRefNotFound, got %v", err)
	}
	// Hash refs work: archive the init commit by short hash.
	if _, err := Archive(cfg, owner, name, "main", "tar.gz"); err != nil {
		t.Errorf("archive by branch: %v", err)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
