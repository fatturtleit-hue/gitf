package handlers

import (
	"net/http"

	"gitf/internal/issues"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type LabelHandler struct {
	Repos  repo.RepositoryStore
	Issues issues.Store
}

func NewLabelHandler(db *gorm.DB, repos repo.RepositoryStore) *LabelHandler {
	return &LabelHandler{Repos: repos, Issues: issues.NewService(db)}
}

func (h *LabelHandler) ListLabels(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	labels, err := h.Issues.ListLabels(repository.ID)
	if err != nil {
		writeServerError(w, r, "list labels", err)
		return
	}
	items := make([]models.LabelDTO, len(labels))
	for i, l := range labels {
		items[i] = l.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *LabelHandler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Color == "" {
		req.Color = "#6b7280"
	}
	label, err := h.Issues.CreateLabel(repository.ID, req.Name, req.Color)
	if err != nil {
		writeServerError(w, r, "create label", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, label.DTO())
}

func (h *LabelHandler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok {
		return
	}
	_, ok = repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	labelID := atoiOrZero(chi.URLParam(r, "labelID"))
	if err := h.Issues.DeleteLabel(uint(labelID)); err != nil {
		writeServerError(w, r, "delete label", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
