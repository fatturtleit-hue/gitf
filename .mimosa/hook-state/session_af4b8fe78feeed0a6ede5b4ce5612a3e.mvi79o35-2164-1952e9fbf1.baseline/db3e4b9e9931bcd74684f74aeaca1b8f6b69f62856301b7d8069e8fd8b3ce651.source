package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gitf/internal/config"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ReleaseHandler holds release HTTP handlers.
type ReleaseHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
	DB     *gorm.DB
}

func NewReleaseHandler(db *gorm.DB, cfg *config.Config, repos repo.RepositoryStore) *ReleaseHandler {
	return &ReleaseHandler{DB: db, Config: cfg, Repos: repos}
}

type CreateReleaseRequest struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	IsDraft    bool   `json:"is_draft"`
	IsPrerelease bool `json:"is_prerelease"`
}

func (h *ReleaseHandler) CreateRelease(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	var req CreateReleaseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	rel := models.Release{
		RepoID:          repository.ID,
		TagName:         req.TagName,
		Name:            req.Name,
		Body:            req.Body,
		IsDraft:         req.IsDraft,
		IsPrerelease:    req.IsPrerelease,
		AuthorID:        user.ID,
	}
	if rel.Name == "" {
		rel.Name = req.TagName
	}
	if err := h.DB.Create(&rel).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeAPIError(w, r, http.StatusConflict, ErrCodeConflict, "release already exists for this tag")
			return
		}
		writeServerError(w, r, "create release", err)
		return
	}
	rel.Author = *user
	writeJSON(w, r, http.StatusCreated, rel.DTO())
}

func (h *ReleaseHandler) ListReleases(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	var releases []models.Release
	if err := h.DB.Preload("Author").Where("repo_id = ?", repository.ID).
		Order("created_at DESC").Find(&releases).Error; err != nil {
		writeServerError(w, r, "list releases", err)
		return
	}
	items := make([]models.ReleaseDTO, len(releases))
	for i, rel := range releases {
		items[i] = rel.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *ReleaseHandler) GetRelease(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	releaseID := atoiOrZero(chi.URLParam(r, "releaseID"))
	var rel models.Release
	if err := h.DB.Preload("Author").Where("repo_id = ? AND id = ?", repository.ID, releaseID).
		First(&rel).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "release not found")
		return
	}
	writeJSON(w, r, http.StatusOK, rel.DTO())
}

func (h *ReleaseHandler) DeleteRelease(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	releaseID := atoiOrZero(chi.URLParam(r, "releaseID"))
	var rel models.Release
	if err := h.DB.Where("repo_id = ? AND id = ?", repository.ID, releaseID).First(&rel).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "release not found")
		return
	}
	// Remove stored assets.
	storageDir := releaseStorageDir(h.Config, repository.ID, rel.ID)
	os.RemoveAll(storageDir)
	h.DB.Delete(&rel)
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReleaseHandler) UploadAsset(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	releaseID := atoiOrZero(chi.URLParam(r, "releaseID"))
	var rel models.Release
	if err := h.DB.Where("repo_id = ? AND id = ?", repository.ID, releaseID).First(&rel).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "release not found")
		return
	}

	// Parse multipart file (10 MB cap).
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "file too large")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "file is required")
		return
	}
	defer file.Close()

	storageDir := releaseStorageDir(h.Config, repository.ID, rel.ID)
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		writeServerError(w, r, "create storage dir", err)
		return
	}
	// Defense against path traversal: header.Filename is fully client-controlled
	// and could be "../../etc/passwd" or contain nested separators. Strip it
	// down to its base name before joining, and reject empty results.
	safeName := filepath.Base(filepath.Clean(header.Filename))
	if safeName == "." || safeName == string(filepath.Separator) || safeName == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid filename")
		return
	}
	storagePath := filepath.Join(storageDir, safeName)
	// Final belt-and-suspenders check: ensure the resolved path is still
	// inside storageDir after cleaning.
	if !isPathInside(storagePath, storageDir) {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid filename")
		return
	}
	dst, err := os.Create(storagePath)
	if err != nil {
		writeServerError(w, r, "create asset file", err)
		return
	}
	written, err := dst.ReadFrom(file)
	dst.Close()
	if err != nil {
		writeServerError(w, r, "write asset", err)
		return
	}

	asset := models.ReleaseAsset{
		ReleaseID:   rel.ID,
		Name:        safeName,
		Size:        written,
		ContentType: header.Header.Get("Content-Type"),
		StoragePath: storagePath,
	}
	if err := h.DB.Create(&asset).Error; err != nil {
		os.Remove(storagePath)
		writeServerError(w, r, "create asset", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, asset.DTO())
}

// releaseStorageDir returns the base directory for release assets.
func releaseStorageDir(cfg *config.Config, repoID, releaseID uint) string {
	return filepath.Join(cfg.DataDir, "releases", fmt.Sprintf("%d", repoID), fmt.Sprintf("%d", releaseID))
}

// repoFromAPIHandler is a package-private helper shared by release and protection handlers.
func repoFromAPIHandler(store repo.RepositoryStore, w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	h := &APIHandler{Config: &config.Config{}, Repos: store}
	return h.repoFromRequestWithRead(w, r)
}
