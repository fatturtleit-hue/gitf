package handlers

import (
	"net/http"

	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ProtectionHandler holds branch protection HTTP handlers.
type ProtectionHandler struct {
	Repos repo.RepositoryStore
	DB    *gorm.DB
}

func NewProtectionHandler(db *gorm.DB, repos repo.RepositoryStore) *ProtectionHandler {
	return &ProtectionHandler{DB: db, Repos: repos}
}

type CreateProtectionRequest struct {
	Pattern              string `json:"pattern"`
	RequireApprovals     int    `json:"require_approvals"`
	AllowForcePush       bool   `json:"allow_force_push"`
	AllowDelete          bool   `json:"allow_delete"`
	RequiredStatusChecks string `json:"required_status_checks"`
	EnforceAdmins        bool   `json:"enforce_admins"`
}

func (h *ProtectionHandler) CreateProtection(w http.ResponseWriter, r *http.Request) {
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
	var req CreateProtectionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	bp := models.BranchProtection{
		RepoID:               repository.ID,
		Pattern:              req.Pattern,
		RequireApprovals:     req.RequireApprovals,
		AllowForcePush:       req.AllowForcePush,
		AllowDelete:          req.AllowDelete,
		RequiredStatusChecks: req.RequiredStatusChecks,
		EnforceAdmins:        req.EnforceAdmins,
	}
	if err := h.DB.Create(&bp).Error; err != nil {
		writeServerError(w, r, "create protection", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, bp.DTO())
}

func (h *ProtectionHandler) ListProtections(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromAPIHandler(h.Repos, w, r)
	if !ok {
		return
	}
	var rules []models.BranchProtection
	if err := h.DB.Where("repo_id = ?", repository.ID).Find(&rules).Error; err != nil {
		writeServerError(w, r, "list protections", err)
		return
	}
	items := make([]models.BranchProtectionDTO, len(rules))
	for i, r := range rules {
		items[i] = r.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *ProtectionHandler) DeleteProtection(w http.ResponseWriter, r *http.Request) {
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
	protID := atoiOrZero(chi.URLParam(r, "protID"))
	if err := h.DB.Where("repo_id = ? AND id = ?", repository.ID, protID).Delete(&models.BranchProtection{}).Error; err != nil {
		writeServerError(w, r, "delete protection", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
