package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/metrics"
	"gitf/internal/models"
	"gitf/internal/repo"
)

// baseEnv captures the process environment once at startup.
var baseEnv []string

func init() {
	baseEnv = append(os.Environ(), "GIT_HTTP_EXPORT_ALL=1")
}

// Sentinel errors returned by checkAccess. Compared with errors.Is rather than
// == — fmt.Errorf produces non-comparable wrapped errors so the previous code
// would have silently broken the moment anyone added wrapping.
var (
	errAuthRequired = errors.New("auth required")
	errAccessDenied = errors.New("access denied")
)

// GitHTTPHandler serves Git smart HTTP protocol.
type GitHTTPHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
}

// NewGitHTTPHandler creates a new Git HTTP handler.
func NewGitHTTPHandler(cfg *config.Config, repos repo.RepositoryStore) *GitHTTPHandler {
	return &GitHTTPHandler{Config: cfg, Repos: repos}
}

// resolveRepo extracts owner+name from URL params, validates them, loads the
// repository, and verifies access. Returns (repository, repoPath, ok). On any
// failure it has already written a 4xx response.
//
// Centralizes the prelude that InfoRefs and serveRPC previously implemented
// independently — there had been four near-identical validation+lookup blocks.
func (h *GitHTTPHandler) resolveRepo(w http.ResponseWriter, r *http.Request, service string) (*models.Repository, string, bool) {
	owner := chi.URLParam(r, "owner")
	name := chi.URLParam(r, "name")
	if !gitutil.ValidateName(owner) || !gitutil.ValidateName(name) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid repository name")
		return nil, "", false
	}

	repository, err := h.Repos.GetByOwnerAndName(owner, name)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
			return nil, "", false
		}
		writeServerError(w, r, "load repo", err)
		return nil, "", false
	}

	user, _ := UserFromContext(r.Context())
	if err := h.checkAccess(repository, user, service); err != nil {
		switch {
		case errors.Is(err, errAuthRequired):
			requireAuth(w, r)
		case errors.Is(err, errAccessDenied):
			writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		default:
			writeServerError(w, r, "check access", err)
		}
		return nil, "", false
	}

	repoPath := gitutil.RepoPath(h.Config, owner, name)
	if _, err := os.Stat(repoPath); err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
		return nil, "", false
	}
	return repository, repoPath, true
}

// checkAccess verifies that the authenticated user has permission for the requested git service.
func (h *GitHTTPHandler) checkAccess(repository *models.Repository, user *models.User, service string) error {
	switch service {
	case "git-receive-pack":
		if !h.Repos.CanWrite(repository, user) {
			if user == nil {
				return errAuthRequired
			}
			return errAccessDenied
		}
	case "git-upload-pack":
		if !h.Repos.CanRead(repository, user) {
			if user == nil {
				return errAuthRequired
			}
			return errAccessDenied
		}
	}
	return nil
}

// InfoRefs handles git-upload-pack / git-receive-pack advertisement.
func (h *GitHTTPHandler) InfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "unknown service")
		return
	}

	_, repoPath, ok := h.resolveRepo(w, r, service)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	pkt := fmt.Sprintf("%04x# service=%s\n", 4+len("# service="+service+"\n"), service)
	if _, err := w.Write([]byte(pkt)); err != nil && !isClientGone(err) {
		writeServerError(w, r, "write info-refs preamble", err)
		return
	}
	if _, err := w.Write([]byte("0000")); err != nil && !isClientGone(err) {
		writeServerError(w, r, "write info-refs flush", err)
		return
	}

	metrics.IncActiveGitOps()
	defer metrics.DecActiveGitOps()

	if err := runGitUploadPack(r, w, repoPath, true); err != nil {
		// Logging happens inside runGit — just return.
		// Set status to indicate server error, even though response has already started.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// UploadPack handles git fetch/clone.
func (h *GitHTTPHandler) UploadPack(w http.ResponseWriter, r *http.Request) {
	h.serveRPC(w, r, "git-upload-pack")
}

// ReceivePack handles git push.
func (h *GitHTTPHandler) ReceivePack(w http.ResponseWriter, r *http.Request) {
	h.serveRPC(w, r, "git-receive-pack")
}

func (h *GitHTTPHandler) serveRPC(w http.ResponseWriter, r *http.Request, service string) {
	_, repoPath, ok := h.resolveRepo(w, r, service)
	if !ok {
		return
	}

	expectedType := "application/x-" + service + "-request"
	if r.Header.Get("Content-Type") != expectedType {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid content type")
		return
	}

	metrics.IncActiveGitOps()
	defer metrics.DecActiveGitOps()

	w.Header().Set("Content-Type", "application/x-"+service+"-result")
	w.WriteHeader(http.StatusOK)

	var err error
	if service == "git-receive-pack" {
		err = runGitReceivePack(r, w, repoPath)
	} else {
		err = runGitUploadPack(r, w, repoPath, false)
	}
	if err != nil {
		// Body partially written — same constraint as InfoRefs.
		// Set status to indicate server error, even though response has already started.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func requireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="gitf"`)
	writeAPIError(w, r, http.StatusUnauthorized, ErrCodeUnauthorized, "authorization required")
}
