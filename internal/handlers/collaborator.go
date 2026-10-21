package handlers

import (
	"net/http"
	"strings"

	"gitf/internal/config"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
)

// CollaboratorHandler manages repo collaborators.
type CollaboratorHandler struct {
	Config *config.Config
	Repos  repo.RepositoryStore
}

// NewCollaboratorHandler creates a collaborator handler.
func NewCollaboratorHandler(cfg *config.Config, store repo.RepositoryStore) *CollaboratorHandler {
	return &CollaboratorHandler{Config: cfg, Repos: store}
}

// requireRepoOwner loads the repo and verifies the current user is the owner.
func (h *CollaboratorHandler) requireRepoOwner(w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	repository, ok := (&APIHandler{Config: h.Config, Repos: h.Repos}).repoFromRequest(w, r)
	if !ok {
		return nil, false
	}
	user, _ := UserFromContext(r.Context())
	if user == nil || user.ID != repository.OwnerID {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "only the repository owner can manage collaborators")
		return nil, false
	}
	return repository, true
}

// ListCollaborators returns all collaborators for a repository.
func (h *CollaboratorHandler) ListCollaborators(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.requireRepoOwner(w, r)
	if !ok {
		return
	}
	collabs, err := h.Repos.ListCollaborators(repository.ID)
	if err != nil {
		writeServerError(w, r, "list collaborators", err)
		return
	}
	dtos := make([]models.CollaboratorDTO, len(collabs))
	for i, c := range collabs {
		dtos[i] = c.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": dtos})
}

type addCollaboratorRequest struct {
	Username   string `json:"username"`
	Permission string `json:"permission"`
	QuotaBytes int64  `json:"quota_bytes"`
}

// AddCollaborator adds a new collaborator to a repository.
func (h *CollaboratorHandler) AddCollaborator(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.requireRepoOwner(w, r)
	if !ok {
		return
	}
	var req addCollaboratorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "username is required")
		return
	}
	if req.Permission == "" {
		req.Permission = "read"
	}
	if req.Permission != "read" && req.Permission != "write" && req.Permission != "admin" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "permission must be read, write, or admin")
		return
	}
	collab, err := h.Repos.AddCollaborator(repository.ID, req.Username, req.Permission, req.QuotaBytes)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	writeJSON(w, r, http.StatusCreated, collab.DTO())
}

type updateCollaboratorRequest struct {
	Permission string `json:"permission"`
	QuotaBytes int64  `json:"quota_bytes"`
}

// UpdateCollaborator updates a collaborator's permission or quota.
func (h *CollaboratorHandler) UpdateCollaborator(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireRepoOwner(w, r)
	if !ok {
		return
	}
	collabID := atoiOrZero(chi.URLParam(r, "collabID"))
	if collabID == 0 {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid collaborator id")
		return
	}
	var req updateCollaboratorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Permission != "" && req.Permission != "read" && req.Permission != "write" && req.Permission != "admin" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "permission must be read, write, or admin")
		return
	}
	collab, err := h.Repos.UpdateCollaborator(uint(collabID), req.Permission, req.QuotaBytes)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "collaborator not found")
		return
	}
	writeJSON(w, r, http.StatusOK, collab.DTO())
}

// RemoveCollaborator removes a collaborator from a repository.
func (h *CollaboratorHandler) RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	repository, ok := h.requireRepoOwner(w, r)
	if !ok {
		return
	}
	userID := atoiOrZero(chi.URLParam(r, "userID"))
	if userID == 0 {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid user id")
		return
	}
	if err := h.Repos.RemoveCollaborator(repository.ID, uint(userID)); err != nil {
		writeServerError(w, r, "remove collaborator", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
