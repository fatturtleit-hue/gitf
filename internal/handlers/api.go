package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// APIHandler holds REST API handlers. Depends only on interfaces — never on
// *gorm.DB directly — so handlers are unit-testable with simple in-memory mocks.
type APIHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
}

// NewAPIHandler creates an API handler backed by the default GORM-based service.
func NewAPIHandler(db *gorm.DB, cfg *config.Config) *APIHandler {
	return &APIHandler{
		Config: cfg,
		Repos:  repo.NewService(db, cfg),
	}
}

// NewAPIHandlerWithStore creates an API handler with an injected repository store
// (used by tests and any caller that wants to swap implementations).
func NewAPIHandlerWithStore(cfg *config.Config, store repo.RepositoryStore) *APIHandler {
	return &APIHandler{Config: cfg, Repos: store}
}

// ── Request parameter sanitization ──────────────────────────────────────────
//
// Branch/tag/ref names and in-repo paths reach git subprocess arguments and
// go-git tree lookups. Each sanitizer validates against a per-character
// allowlist and returns a freshly built copy, so no unvalidated request input
// ever flows into a git operation.

func refCharOK(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
		r == '.' || r == '_' || r == '-' || r == '/'
}

// sanitizeRefParam validates a client-supplied branch/tag/commit parameter.
func sanitizeRefParam(raw string) (string, bool) {
	if raw == "" || len(raw) > 200 {
		return "", false
	}
	var b strings.Builder
	for _, r := range raw {
		if !refCharOK(r) {
			return "", false
		}
		b.WriteRune(r)
	}
	s := b.String()
	if strings.Contains(s, "..") || strings.HasPrefix(s, "-") {
		return "", false
	}
	return s, true
}

// sanitizeRepoTreePath validates a client-supplied path inside a repository
// tree (tree/blob APIs). Empty means the tree root.
func sanitizeRepoTreePath(raw string) (string, bool) {
	if raw == "" {
		return "", true
	}
	if len(raw) > 1024 || strings.HasPrefix(raw, "-") || strings.HasPrefix(raw, "/") {
		return "", false
	}
	var b strings.Builder
	for i, seg := range strings.Split(raw, "/") {
		if seg == "" || seg == "." || seg == ".." || len(seg) > 255 {
			return "", false
		}
		for _, r := range seg {
			if !refCharOK(r) && r != ' ' && r != '(' && r != ')' {
				return "", false
			}
		}
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(seg)
	}
	return b.String(), true
}

// CreateRepoRequest is the body for repository creation.
type CreateRepoRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	IsPrivate     bool   `json:"is_private"`
	DefaultBranch string `json:"default_branch"`
}

// repoFromRequest extracts {owner}/{name} from chi URL params, validates them,
// and loads the repository. On any failure it has already written a 400/404
// response and returns false. All repo-scoped handlers funnel through this.
func (h *APIHandler) repoFromRequest(w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	owner := chi.URLParam(r, "owner")
	name := chi.URLParam(r, "name")
	if !gitutil.ValidateName(owner) || !gitutil.ValidateName(name) {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
		return nil, false
	}
	repository, err := h.Repos.GetByOwnerAndName(owner, name)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
			return nil, false
		}
		writeServerError(w, r, "load repo", err)
		return nil, false
	}
	return repository, true
}

// repoFromRequestWithRead is repoFromRequest plus a read-permission check.
func (h *APIHandler) repoFromRequestWithRead(w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	repository, ok := h.repoFromRequest(w, r)
	if !ok {
		return nil, false
	}
	user, _ := UserFromContext(r.Context())
	if !h.Repos.CanRead(repository, user) {
		// Same code/message as not-found to avoid leaking the existence
		// of private repos to unauthenticated callers.
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
		return nil, false
	}
	return repository, true
}

// branchOrDefault returns the ?branch= query value, defaulting to repo's default branch.
// Centralized so the same fallback logic isn't repeated in every git handler.
func branchOrDefault(r *http.Request, repository *models.Repository) string {
	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branch = repository.DefaultBranch
	}
	return branch
}

// ListRepos lists repositories visible to the current user.
func (h *APIHandler) ListRepos(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFromContext(r.Context())
	var userID uint
	if user != nil {
		userID = user.ID
	}

	search := r.URL.Query().Get("search")
	page := atoiOrZero(r.URL.Query().Get("page"))
	perPage := atoiOrZero(r.URL.Query().Get("per_page"))
	page, perPage = clampPage(page, perPage)
	offset := (page - 1) * perPage

	repos, err := h.Repos.List(userID, search, perPage, offset)
	if err != nil {
		writeServerError(w, r, "list repos", err)
		return
	}

	total, err := h.Repos.Count(userID, search)
	if err != nil {
		writeServerError(w, r, "count repos", err)
		return
	}

	items := make([]models.RepoDTO, len(repos))
	for i, rep := range repos {
		items[i] = rep.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": NewPagination(page, perPage, total),
	})
}

// CreateRepo creates a new repository.
func (h *APIHandler) CreateRepo(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	var req CreateRepoRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "name is required")
		return
	}
	if !gitutil.ValidateName(req.Name) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid repository name")
		return
	}

	repository, err := h.Repos.Create(repo.OwnerTypeUser, user.ID, req.Name, req.Description, req.DefaultBranch, req.IsPrivate)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrInvalidName):
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid repository name")
		case errors.Is(err, repo.ErrOwnerNotFound):
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "owner not found")
		case errors.Is(err, repo.ErrConflict):
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "repository already exists")
		default:
			writeServerError(w, r, "create repo", err)
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, repository.DTO())
}

// GetRepo returns a single repository.
func (h *APIHandler) GetRepo(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	writeJSON(w, r, http.StatusOK, repository.DTO())
}

// DeleteRepo deletes a repository.
func (h *APIHandler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := h.repoFromRequest(w, r)
	if !ok {
		return
	}
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	if err := h.Repos.Delete(repository.OwnerType, repository.OwnerID, repository.Name); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "repository not found")
			return
		}
		writeServerError(w, r, "delete repo", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRepoStats returns aggregate repo statistics.
func (h *APIHandler) GetRepoStats(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branch := branchOrDefault(r, repository)
	stats, err := gitutil.Stats(h.Config, repository.Owner.Username, repository.Name, branch)
	if err != nil {
		writeServerError(w, r, "repo stats", err)
		return
	}
	writeJSON(w, r, http.StatusOK, stats)
}

// GetRepoBranches returns branches for a repository.
func (h *APIHandler) GetRepoBranches(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branches, err := gitutil.Branches(h.Config, repository.Owner.Username, repository.Name)
	if err != nil {
		writeServerError(w, r, "list branches", err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"branches": branches,
		"default":  repository.DefaultBranch,
	})
}

// GetRepoCommits returns commit history.
func (h *APIHandler) GetRepoCommits(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branch := branchOrDefault(r, repository)
	limit := atoiOrZero(r.URL.Query().Get("limit"))
	commits, err := gitutil.Commits(h.Config, repository.Owner.Username, repository.Name, branch, limit)
	if err != nil {
		if errors.Is(err, gitutil.ErrRepoEmpty) {
			writeJSON(w, r, http.StatusOK, map[string]any{"commits": []gitutil.CommitInfo{}})
			return
		}
		writeServerError(w, r, "list commits", err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"commits": commits})
}

// GetRepoCommit returns a single commit by hash.
func (h *APIHandler) GetRepoCommit(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	hash, ok := sanitizeRefParam(chi.URLParam(r, "hash"))
	if !ok || hash == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "hash is required")
		return
	}
	commit, err := gitutil.Commit(h.Config, repository.Owner.Username, repository.Name, hash)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "commit not found")
		return
	}
	writeJSON(w, r, http.StatusOK, commit)
}

// GetRepoTree returns directory entries.
func (h *APIHandler) GetRepoTree(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branch := branchOrDefault(r, repository)
	path, ok := sanitizeRepoTreePath(r.URL.Query().Get("path"))
	if !ok {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "path not found")
		return
	}
	entries, err := gitutil.TreeEntries(h.Config, repository.Owner.Username, repository.Name, branch, path)
	if err != nil {
		if errors.Is(err, gitutil.ErrRepoEmpty) {
			writeJSON(w, r, http.StatusOK, map[string]any{"entries": []gitutil.TreeEntry{}})
			return
		}
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "path not found")
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"entries": entries})
}

// GetRepoBlob returns blob metadata and content.
func (h *APIHandler) GetRepoBlob(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branch := branchOrDefault(r, repository)
	path, ok := sanitizeRepoTreePath(r.URL.Query().Get("path"))
	if !ok {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "file not found")
		return
	}
	content, err := gitutil.BlobContent(h.Config, repository.Owner.Username, repository.Name, branch, path)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "file not found")
		return
	}
	size, err := gitutil.BlobSize(h.Config, repository.Owner.Username, repository.Name, branch, path)
	if err != nil {
		// Don't fail the whole request — content is the primary payload.
		// Log so investigators can correlate by request_id.
		writeServerError(w, r, "blob size lookup", err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"content": string(content),
		"size":    size,
		"path":    path,
	})
}

// GetRepoRaw returns raw blob content.
func (h *APIHandler) GetRepoRaw(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	branch := branchOrDefault(r, repository)
	path, ok := sanitizeRepoTreePath(r.URL.Query().Get("path"))
	if !ok {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "file not found")
		return
	}
	content, err := gitutil.BlobContent(h.Config, repository.Owner.Username, repository.Name, branch, path)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := w.Write([]byte(content)); err != nil && !isClientGone(err) {
		writeServerError(w, r, "write raw blob", err)
	}
}

// ── Phase 1 new endpoints ──────────────────────────

// GetRepoTags returns all tags for a repository.
func (h *APIHandler) GetRepoTags(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	tags, err := gitutil.Tags(h.Config, repository.Owner.Username, repository.Name)
	if err != nil {
		writeServerError(w, r, "list tags", err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"tags": tags})
}

// GetRepoCompare returns a diff between two branches/commits.
func (h *APIHandler) GetRepoCompare(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	base, okB := sanitizeRefParam(chi.URLParam(r, "base"))
	head, okH := sanitizeRefParam(chi.URLParam(r, "head"))
	if !okB || !okH {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid ref")
		return
	}
	diffs, err := gitutil.Diff(h.Config, repository.Owner.Username, repository.Name, base, head)
	if err != nil {
		writeServerError(w, r, "compare diff", err)
		return
	}
	stats, _ := gitutil.DiffStat(h.Config, repository.Owner.Username, repository.Name, base, head)
	writeJSON(w, r, http.StatusOK, map[string]any{"diffs": diffs, "stats": stats})
}

// GetRepoArchive streams a git archive of the repository at the given ref.
func (h *APIHandler) GetRepoArchive(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.repoFromRequestWithRead(w, r)
	if !ok {
		return
	}
	ref, ok := sanitizeRefParam(chi.URLParam(r, "ref"))
	if !ok {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "ref not found")
		return
	}
	format := r.URL.Query().Get("format")
	if format != "zip" {
		format = "tar.gz"
	}
	reader, err := gitutil.Archive(h.Config, repository.Owner.Username, repository.Name, ref, format)
	if err != nil {
		if errors.Is(err, gitutil.ErrRefNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "ref not found")
			return
		}
		writeServerError(w, r, "create archive", err)
		return
	}
	defer reader.Close()
	contentType := "application/gzip"
	if format == "zip" {
		contentType = "application/zip"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, reader); err != nil && !isClientGone(err) {
		writeServerError(w, r, "stream archive", err)
	}
}

// CreateForkRequest is the body for forking a repository.
type CreateForkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateFork forks a repository under the authenticated user.
func (h *APIHandler) CreateFork(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	source, ok := h.repoFromRequest(w, r)
	if !ok {
		return
	}
	var req CreateForkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		req.Name = source.Name
	}
	if !gitutil.ValidateName(req.Name) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid repository name")
		return
	}
	fork, err := h.Repos.Fork(repo.OwnerTypeUser, user.ID, req.Name, req.Description, source.DefaultBranch, source.IsPrivate, source.ID)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrOwnerNotFound):
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "owner not found")
		case errors.Is(err, repo.ErrConflict):
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "repository already exists")
		default:
			writeServerError(w, r, "fork repo", err)
		}
		return
	}
	writeJSON(w, r, http.StatusCreated, fork.DTO())
}

// UpdateRepoRequest is the body for updating repository settings.
type UpdateRepoRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	IsPrivate     *bool   `json:"is_private,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
}

// UpdateRepo updates a repository's settings.
func (h *APIHandler) UpdateRepo(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := h.repoFromRequest(w, r)
	if !ok {
		return
	}
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	var req UpdateRepoRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name != nil {
		if !gitutil.ValidateName(*req.Name) {
			writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid repository name")
			return
		}
		repository.Name = *req.Name
	}
	if req.Description != nil {
		repository.Description = *req.Description
	}
	if req.IsPrivate != nil {
		repository.IsPrivate = *req.IsPrivate
	}
	if req.DefaultBranch != nil {
		repository.DefaultBranch = *req.DefaultBranch
	}
	if err := h.Repos.Update(repository); err != nil {
		writeServerError(w, r, "update repo", err)
		return
	}
	writeJSON(w, r, http.StatusOK, repository.DTO())
}
