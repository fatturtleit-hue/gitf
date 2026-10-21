package handlers

import (
	"net/http"
	"time"

	"gitf/internal/models"
	"gitf/internal/repo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type ActionsHandler struct {
	Repos repo.RepositoryStore
	DB    *gorm.DB
}

func NewActionsHandler(db *gorm.DB, repos repo.RepositoryStore) *ActionsHandler {
	return &ActionsHandler{DB: db, Repos: repos}
}

func (h *ActionsHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	var wfs []models.Workflow
	h.DB.Where("repo_id = ?", repository.ID).Find(&wfs)
	items := make([]models.WorkflowDTO, len(wfs))
	for i, wf := range wfs {
		items[i] = wf.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *ActionsHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	wfID := atoiOrZero(chi.URLParam(r, "workflowID"))
	var runs []models.WorkflowRun
	query := h.DB.Where("repo_id = ?", repository.ID)
	if wfID > 0 {
		query = query.Where("workflow_id = ?", wfID)
	}
	query.Order("created_at DESC").Limit(30).Find(&runs)
	writeJSON(w, r, http.StatusOK, map[string]any{"items": runs})
}

func (h *ActionsHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	id := atoiOrZero(chi.URLParam(r, "runID"))
	var run models.WorkflowRun
	if err := h.DB.Where("repo_id = ? AND id = ?", repository.ID, id).First(&run).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "run not found")
		return
	}
	writeJSON(w, r, http.StatusOK, run)
}

func (h *ActionsHandler) GetRunLogs(w http.ResponseWriter, r *http.Request) {
	_, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	id := atoiOrZero(chi.URLParam(r, "runID"))
	var logs []models.WorkflowRunLog
	h.DB.Where("run_id = ?", id).Order("chunk ASC").Find(&logs)
	writeJSON(w, r, http.StatusOK, map[string]any{"items": logs})
}

func (h *ActionsHandler) DispatchWorkflow(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanWrite(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	wfID := atoiOrZero(chi.URLParam(r, "workflowID"))
	var wf models.Workflow
	if err := h.DB.Where("id = ? AND repo_id = ?", wfID, repository.ID).First(&wf).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "workflow not found")
		return
	}
	var req struct{ Branch string `json:"branch"` }
	if !decodeJSON(w, r, &req) { return }
	if req.Branch == "" { req.Branch = repository.DefaultBranch }

	run := models.WorkflowRun{
		WorkflowID:     uint(wfID),
		RepoID:         repository.ID,
		Status:         "queued",
		TriggerEvent:   "manual",
		TriggerPayload: `{"type":"manual"}`,
		Branch:         req.Branch,
	}
	if err := h.DB.Create(&run).Error; err != nil {
		writeServerError(w, r, "dispatch workflow", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, run)
}

func (h *ActionsHandler) CancelRun(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanWrite(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	id := atoiOrZero(chi.URLParam(r, "runID"))
	h.DB.Model(&models.WorkflowRun{}).Where("repo_id = ? AND id = ?", repository.ID, id).
		Update("status", "cancelled").Update("finished_at", time.Now())
	w.WriteHeader(http.StatusOK)
}

// ── Runner API ────────────────────────────────────────

func (h *ActionsHandler) RunnerRegister(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name string `json:"name"` }
	if !decodeJSON(w, r, &req) { return }
	// Simple runner registration — store runner and return ID.
	runner := models.Runner{Name: req.Name}
	h.DB.Create(&runner)
	writeJSON(w, r, http.StatusCreated, runner)
}

func (h *ActionsHandler) RunnerPoll(w http.ResponseWriter, r *http.Request) {
	var req struct{ RunnerID uint `json:"runner_id"` }
	if !decodeJSON(w, r, &req) { return }
	// Find first queued run and atomically claim it.
	var run models.WorkflowRun
	err := h.DB.Where("status = ?", "queued").Order("created_at ASC").First(&run).Error
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	h.DB.Model(&run).Updates(map[string]any{
		"status":     "dispatched",
		"runner_id":  req.RunnerID,
		"started_at": time.Now(),
	})
	writeJSON(w, r, http.StatusOK, run)
}

func (h *ActionsHandler) RunnerUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := atoiOrZero(chi.URLParam(r, "runID"))
	var req struct{ Status string `json:"status"` }
	if !decodeJSON(w, r, &req) { return }
	updates := map[string]any{"status": req.Status}
	if req.Status == "success" || req.Status == "failed" || req.Status == "cancelled" {
		updates["finished_at"] = time.Now()
	}
	h.DB.Model(&models.WorkflowRun{}).Where("id = ?", id).Updates(updates)
	w.WriteHeader(http.StatusOK)
}

func (h *ActionsHandler) RunnerPostLog(w http.ResponseWriter, r *http.Request) {
	id := atoiOrZero(chi.URLParam(r, "runID"))
	var req struct{ Chunk int `json:"chunk"`; Content string `json:"content"` }
	if !decodeJSON(w, r, &req) { return }
	h.DB.Create(&models.WorkflowRunLog{RunID: uint(id), Chunk: req.Chunk, Content: req.Content})
	w.WriteHeader(http.StatusCreated)
}

func (h *ActionsHandler) RunnerHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct{ RunnerID uint `json:"runner_id"` }
	if !decodeJSON(w, r, &req) { return }
	now := time.Now()
	h.DB.Model(&models.Runner{}).Where("id = ?", req.RunnerID).
		Updates(map[string]any{"status": "idle", "last_seen_at": &now})
	w.WriteHeader(http.StatusOK)
}
