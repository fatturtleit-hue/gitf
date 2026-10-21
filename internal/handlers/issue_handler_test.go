package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitf/internal/models"
)

// issueRequest runs an issue-scoped handler call; call is the bound handler
// method (e.g. h.CreateIssue). Route params owner/name always target alice/proj.
func issueRequest(t *testing.T, env *htEnv, h *IssueHandler, user *models.User, params map[string]string, body any, call func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	if params == nil {
		params = map[string]string{}
	}
	params["owner"] = "alice"
	params["name"] = "proj"
	req := chiContext(t, newTestRequest(t, http.MethodGet, "/api/v1", body), params)
	if user != nil {
		req = asUser(req, user)
	}
	w := httptest.NewRecorder()
	call(w, req)
	return w
}

func TestIssueLifecycle(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	bob := env.user(t, "bob")
	env.seedRepo(t, "proj", false) // public repo — read access for everyone
	h := NewIssueHandler(env.db, env.repos)

	// Any authenticated reader of a public repo can open issues (by design).
	w := issueRequest(t, env, h, bob, nil, map[string]string{"title": "login broken", "body": "it is"}, h.CreateIssue)
	if w.Code != http.StatusCreated {
		t.Fatalf("create issue: got %d: %s", w.Code, w.Body.String())
	}

	// List includes it.
	w = issueRequest(t, env, h, alice, nil, nil, h.ListIssues)
	if w.Code != http.StatusOK {
		t.Fatalf("list issues: got %d", w.Code)
	}

	// Get by number.
	w = issueRequest(t, env, h, bob, map[string]string{"number": "1"}, nil, h.GetIssue)
	if w.Code != http.StatusOK {
		t.Fatalf("get issue: got %d", w.Code)
	}

	// Comment on it.
	w = issueRequest(t, env, h, alice, map[string]string{"number": "1"}, map[string]string{"body": "looking"}, h.AddComment)
	if w.Code != http.StatusCreated {
		t.Fatalf("add comment: got %d: %s", w.Code, w.Body.String())
	}
	w = issueRequest(t, env, h, bob, map[string]string{"number": "1"}, nil, h.ListComments)
	if w.Code != http.StatusOK {
		t.Fatalf("list comments: got %d", w.Code)
	}

	// Close and reopen.
	w = issueRequest(t, env, h, alice, map[string]string{"number": "1"}, nil, h.CloseIssue)
	if w.Code != http.StatusOK {
		t.Fatalf("close: got %d", w.Code)
	}
	w = issueRequest(t, env, h, alice, map[string]string{"number": "1"}, nil, h.ReopenIssue)
	if w.Code != http.StatusOK {
		t.Fatalf("reopen: got %d", w.Code)
	}

	// Unknown issue number → 404.
	w = issueRequest(t, env, h, alice, map[string]string{"number": "99"}, nil, h.GetIssue)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown issue: got %d, want 404", w.Code)
	}
}

func TestLabelHandlers(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	env.seedRepo(t, "proj", false)
	h := NewLabelHandler(env.db, env.repos)

	req := chiContext(t, asUser(newTestRequest(t, http.MethodPost, "/labels", map[string]string{"name": "bug", "color": "#ff0000"}), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w := httptest.NewRecorder()
	h.CreateLabel(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create label: got %d: %s", w.Code, w.Body.String())
	}

	req = chiContext(t, asUser(newTestRequest(t, http.MethodGet, "/labels", nil), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w = httptest.NewRecorder()
	h.ListLabels(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("list labels: got %d", w.Code)
	}

	req = chiContext(t, asUser(newTestRequest(t, http.MethodDelete, "/labels/1", nil), alice),
		map[string]string{"owner": "alice", "name": "proj", "labelID": "1"})
	w = httptest.NewRecorder()
	h.DeleteLabel(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete label: got %d", w.Code)
	}
}

func TestMilestoneHandlers(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	env.seedRepo(t, "proj", false)
	h := NewMilestoneHandler(env.db, env.repos)

	req := chiContext(t, asUser(newTestRequest(t, http.MethodPost, "/milestones",
		map[string]string{"title": "M1", "description": "first", "due_date": "2026-12-31"}), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w := httptest.NewRecorder()
	h.CreateMilestone(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create milestone: got %d: %s", w.Code, w.Body.String())
	}

	req = chiContext(t, asUser(newTestRequest(t, http.MethodGet, "/milestones", nil), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w = httptest.NewRecorder()
	h.ListMilestones(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("list milestones: got %d", w.Code)
	}

	req = chiContext(t, asUser(newTestRequest(t, http.MethodPost, "/milestones/close",
		map[string]string{"title": "M1"}), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w = httptest.NewRecorder()
	h.CloseMilestone(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("close milestone: got %d: %s", w.Code, w.Body.String())
	}
}

func TestKanbanHandlers(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	env.seedRepo(t, "proj", false)
	ih := NewIssueHandler(env.db, env.repos)
	kh := NewKanbanHandler(env.db, env.repos)

	issueRequest(t, env, ih, alice, nil, map[string]string{"title": "task"}, ih.CreateIssue)

	req := chiContext(t, asUser(newTestRequest(t, http.MethodGet, "/kanban", nil), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w := httptest.NewRecorder()
	kh.GetColumns(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get columns: got %d", w.Code)
	}

	req = chiContext(t, asUser(newTestRequest(t, http.MethodPut, "/kanban/move",
		map[string]any{"issue_id": 1, "column_id": 1, "position": 0}), alice),
		map[string]string{"owner": "alice", "name": "proj"})
	w = httptest.NewRecorder()
	kh.MoveIssue(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("move issue: got %d: %s", w.Code, w.Body.String())
	}
}
