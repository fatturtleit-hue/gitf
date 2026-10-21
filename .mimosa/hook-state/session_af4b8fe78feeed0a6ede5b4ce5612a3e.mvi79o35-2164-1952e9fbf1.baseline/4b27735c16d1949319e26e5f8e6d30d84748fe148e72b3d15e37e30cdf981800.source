package handlers

import (
	"net/http"

	"gitf/internal/issues"
	"gitf/internal/models"
	"gitf/internal/repo"

	"gorm.io/gorm"
)

type MilestoneHandler struct {
	Repos  repo.RepositoryStore
	Issues issues.Store
}

func NewMilestoneHandler(db *gorm.DB, repos repo.RepositoryStore) *MilestoneHandler {
	return &MilestoneHandler{Repos: repos, Issues: issues.NewService(db)}
}

func (h *MilestoneHandler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	milestones, err := h.Issues.ListMilestones(repository.ID)
	if err != nil {
		writeServerError(w, r, "list milestones", err)
		return
	}
	items := make([]models.MilestoneDTO, len(milestones))
	for i, m := range milestones {
		items[i] = m.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *MilestoneHandler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		DueDate     *string `json:"due_date"`
	}
	if !decodeJSON(w, r, &req) { return }
	m, err := h.Issues.CreateMilestone(repository.ID, req.Title, req.Description, req.DueDate)
	if err != nil {
		writeServerError(w, r, "create milestone", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, m.DTO())
}

func (h *MilestoneHandler) CloseMilestone(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	var req struct{ Title string `json:"title"` }
	if !decodeJSON(w, r, &req) { return }
	milestones, err := h.Issues.ListMilestones(repository.ID)
	if err != nil {
		writeServerError(w, r, "list milestones", err)
		return
	}
	var target *models.Milestone
	for i := range milestones {
		if milestones[i].Title == req.Title {
			target = &milestones[i]
			break
		}
	}
	if target == nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "milestone not found")
		return
	}
	if err := h.Issues.CloseMilestone(target); err != nil {
		writeServerError(w, r, "close milestone", err)
		return
	}
	writeJSON(w, r, http.StatusOK, target.DTO())
}
