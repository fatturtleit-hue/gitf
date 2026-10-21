package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"gitf/internal/issues"
	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type IssueHandler struct {
	Repos  repo.RepositoryStore
	Issues issues.Store
	DB     *gorm.DB
}

func NewIssueHandler(db *gorm.DB, repos repo.RepositoryStore) *IssueHandler {
	return &IssueHandler{DB: db, Repos: repos, Issues: issues.NewService(db)}
}

type CreateIssueRequest struct {
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	AssigneeID  *uint   `json:"assignee_id"`
	MilestoneID *uint   `json:"milestone_id"`
	LabelIDs    []uint  `json:"label_ids"`
}

func (h *IssueHandler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	var req CreateIssueRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "title is required")
		return
	}
	issue, err := h.Issues.Create(repository.ID, user.ID, req.Title, req.Body, req.AssigneeID, req.MilestoneID, req.LabelIDs)
	if err != nil {
		writeServerError(w, r, "create issue", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, issue.DTO())
}

func (h *IssueHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	page := atoiOrZero(r.URL.Query().Get("page"))
	perPage := atoiOrZero(r.URL.Query().Get("per_page"))
	page, perPage = clampPage(page, perPage)
	offset := (page - 1) * perPage

	issues, total, err := h.Issues.List(repository.ID,
		r.URL.Query().Get("status"),
		r.URL.Query().Get("label"),
		r.URL.Query().Get("milestone"),
		r.URL.Query().Get("assignee"),
		r.URL.Query().Get("q"),
		perPage, offset)
	if err != nil {
		writeServerError(w, r, "list issues", err)
		return
	}
	items := make([]models.IssueDTO, len(issues))
	for i, iss := range issues {
		items[i] = iss.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"items":      items,
		"pagination": NewPagination(page, perPage, total),
	})
}

func (h *IssueHandler) GetIssue(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid issue number")
		return
	}
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		if errors.Is(err, issues.ErrNotFound) {
			writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
			return
		}
		writeServerError(w, r, "get issue", err)
		return
	}
	writeJSON(w, r, http.StatusOK, issue.DTO())
}

func (h *IssueHandler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok {
		return
	}
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid issue number")
		return
	}
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Body        *string `json:"body"`
		AssigneeID  *uint   `json:"assignee_id"`
		MilestoneID *uint   `json:"milestone_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title != nil { issue.Title = *req.Title }
	if req.Body != nil { issue.Body = *req.Body }
	if req.AssigneeID != nil { issue.AssigneeID = req.AssigneeID }
	if req.MilestoneID != nil { issue.MilestoneID = req.MilestoneID }
	if err := h.Issues.Update(issue); err != nil {
		writeServerError(w, r, "update issue", err)
		return
	}
	writeJSON(w, r, http.StatusOK, issue.DTO())
}

func (h *IssueHandler) CloseIssue(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
		return
	}
	if err := h.Issues.Close(issue, user.ID); err != nil {
		writeServerError(w, r, "close issue", err)
		return
	}
	writeJSON(w, r, http.StatusOK, issue.DTO())
}

func (h *IssueHandler) ReopenIssue(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
		return
	}
	if err := h.Issues.Reopen(issue); err != nil {
		writeServerError(w, r, "reopen issue", err)
		return
	}
	writeJSON(w, r, http.StatusOK, issue.DTO())
}

// ── Comments ───────────────────────────────────────────

func (h *IssueHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
		return
	}
	var req struct{ Body string `json:"body"` }
	if !decodeJSON(w, r, &req) { return }
	if req.Body == "" {
		writeAPIError(w, r, http.StatusBadRequest, ErrCodeValidation, "body is required")
		return
	}
	comment, err := h.Issues.AddComment(repository.ID, issue.ID, user.ID, req.Body)
	if err != nil {
		writeServerError(w, r, "add comment", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, comment.DTO())
}

func (h *IssueHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	number, _ := strconv.Atoi(chi.URLParam(r, "number"))
	issue, err := h.Issues.GetByNumber(repository.ID, number)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "issue not found")
		return
	}
	comments, err := h.Issues.ListComments(issue.ID)
	if err != nil {
		writeServerError(w, r, "list comments", err)
		return
	}
	items := make([]models.CommentDTO, len(comments))
	for i, c := range comments {
		items[i] = c.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func repoFromIssueHandler(store repo.RepositoryStore, w http.ResponseWriter, r *http.Request) (*models.Repository, bool) {
	h := &APIHandler{Repos: store}
	return h.repoFromRequestWithRead(w, r)
}
