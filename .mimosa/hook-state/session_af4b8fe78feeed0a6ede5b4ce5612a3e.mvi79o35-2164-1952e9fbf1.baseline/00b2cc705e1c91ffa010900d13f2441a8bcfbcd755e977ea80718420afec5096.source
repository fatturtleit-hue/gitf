package handlers

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"gitf/internal/web"
)

var (
	spaFS      fs.FS
	indexBytes []byte
	spaFiles   = make(map[string]bool)
	spaOnce    sync.Once
	spaErr     error
)

func initSPA() {
	var err error
	spaFS, err = fs.Sub(web.Dist, "dist")
	if err != nil {
		spaErr = err
		return
	}

	// Pre-load index.html into memory for fast SPA fallback.
	indexBytes, err = fs.ReadFile(spaFS, "index.html")
	if err != nil {
		spaErr = err
		return
	}

	// Build an in-memory set of all embedded files for O(1) lookup.
	if err := fs.WalkDir(spaFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			spaFiles[path] = true
		}
		return nil
	}); err != nil {
		spaErr = err
	}
}

// StaticHandler serves the embedded SPA with optimal caching.
func StaticHandler() http.Handler {
	spaOnce.Do(initSPA)
	if spaErr != nil {
		slog.Error("spa init failed", "error", spaErr)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, r, http.StatusInternalServerError, ErrCodeInternal, "frontend unavailable")
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[1:] // trim leading /

		// Set cache headers based on file type.
		switch filepath.Ext(name) {
		case ".js", ".css", ".ico", ".png", ".jpg", ".svg", ".woff2":
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case ".html":
			w.Header().Set("Cache-Control", "no-cache")
		default:
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}

		// SPA fallback: serve index.html for any path that doesn't match a static file.
		if name == "" || !spaFiles[name] {
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", time.Time{}, &bytesReader{b: indexBytes})
			return
		}

		// Serve static file content from embed FS.
		data, err := fs.ReadFile(spaFS, name)
		if err != nil {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "file not found")
			return
		}
		http.ServeContent(w, r, filepath.Base(name), time.Time{}, &bytesReader{b: data})
	})
}

// bytesReader wraps a byte slice to implement io.ReadSeeker.
type bytesReader struct {
	b      []byte
	offset int64
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.offset >= int64(len(r.b)) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.offset:])
	r.offset += int64(n)
	return n, nil
}

func (r *bytesReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = int64(len(r.b)) + offset
	}
	return r.offset, nil
}
