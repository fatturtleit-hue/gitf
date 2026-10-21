package handlers

import (
	"net/http"

	"gitf/internal/issues"
	"gitf/internal/repo"

	"gorm.io/gorm"
)

type KanbanHandler struct {
	Repos  repo.RepositoryStore
	Issues issues.Store
}

func NewKanbanHandler(db *gorm.DB, repos repo.RepositoryStore) *KanbanHandler {
	return &KanbanHandler{Repos: repos, Issues: issues.NewService(db)}
}

func (h *KanbanHandler) GetColumns(w http.ResponseWriter, r *http.Request) {
	repository, ok := repoFromIssueHandler(h.Repos, w, r)
	if !ok {
		return
	}
	cols, err := h.Issues.GetColumns(repository.ID)
	if err != nil {
		writeServerError(w, r, "get columns", err)
		return
	}
	// For each column, fetch its issues.
	type ColumnWithIssues struct {
		ID       uint                 `json:"id"`
		Name     string               `json:"name"`
		Position int                  `json:"position"`
		Issues   []interface{}        `json:"issues"`
	}
	var result []ColumnWithIssues
	for _, col := range cols {
		cwi := ColumnWithIssues{ID: col.ID, Name: col.Name, Position: col.Position, Issues: []interface{}{}}
		iss, err := h.Issues.GetColumnIssues(col.ID)
		if err == nil {
			for _, i := range iss {
				cwi.Issues = append(cwi.Issues, i.DTO())
			}
		}
		if cwi.Issues == nil {
			cwi.Issues = []interface{}{}
		}
		result = append(result, cwi)
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"columns": result})
}

func (h *KanbanHandler) MoveIssue(w http.ResponseWriter, r *http.Request) {
	_, ok := requireUser(w, r)
	if !ok { return }
	_, ok = repoFromIssueHandler(h.Repos, w, r)
	if !ok { return }
	var req struct {
		IssueID  uint `json:"issue_id"`
		ColumnID uint `json:"column_id"`
		Position int  `json:"position"`
	}
	if !decodeJSON(w, r, &req) { return }
	if err := h.Issues.MoveIssue(req.IssueID, req.ColumnID, req.Position); err != nil {
		writeServerError(w, r, "move issue", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
