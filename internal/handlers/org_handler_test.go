package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitf/internal/models"
)

// runOrg builds an OrgHandler request with chi params and auth context, and
// executes the given handler method.
func runOrg(t *testing.T, env *htEnv, h *OrgHandler, user *models.User, method string, params map[string]string, body any, call func(*OrgHandler, http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := chiContext(t, newTestRequest(t, method, "/api/v1/orgs", body), params)
	if user != nil {
		req = asUser(req, user)
	}
	w := httptest.NewRecorder()
	call(h, w, req)
	return w
}

func TestOrgTeamAuthorizationMatrix(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	bob := env.user(t, "bob")
	h := NewOrgHandler(env.db, env.cfg, env.repos)

	// alice creates the organization.
	w := runOrg(t, env, h, alice, http.MethodPost, nil,
		map[string]string{"name": "acme", "display_name": "Acme"},
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.CreateOrg(w, r) })
	if w.Code != http.StatusCreated {
		t.Fatalf("create org: got %d: %s", w.Code, w.Body.String())
	}

	// alice adds bob as a plain member.
	w = runOrg(t, env, h, alice, http.MethodPost, map[string]string{"org": "acme"},
		map[string]string{"username": "bob", "role": "member"},
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.AddMember(w, r) })
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("add member: got %d: %s", w.Code, w.Body.String())
	}

	// REGRESSION (security): a plain member must NOT be able to create teams.
	w = runOrg(t, env, h, bob, http.MethodPost, map[string]string{"org": "acme"},
		map[string]string{"name": "sneaky"},
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.CreateTeam(w, r) })
	if w.Code != http.StatusForbidden {
		t.Errorf("member create team: got %d, want 403", w.Code)
	}

	// The owner can create teams.
	w = runOrg(t, env, h, alice, http.MethodPost, map[string]string{"org": "acme"},
		map[string]string{"name": "devs"},
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.CreateTeam(w, r) })
	if w.Code != http.StatusCreated {
		t.Fatalf("owner create team: got %d: %s", w.Code, w.Body.String())
	}

	// Listing teams works for the owner.
	w = runOrg(t, env, h, alice, http.MethodGet, map[string]string{"org": "acme"}, nil,
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.ListTeams(w, r) })
	if w.Code != http.StatusOK {
		t.Errorf("list teams: got %d", w.Code)
	}

	// A member must NOT be able to delete a team.
	w = runOrg(t, env, h, bob, http.MethodDelete, map[string]string{"org": "acme", "teamID": "1"}, nil,
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.DeleteTeam(w, r) })
	if w.Code != http.StatusForbidden {
		t.Errorf("member delete team: got %d, want 403", w.Code)
	}

	// A non-owner must NOT be able to delete the organization.
	w = runOrg(t, env, h, bob, http.MethodDelete, map[string]string{"org": "acme"}, nil,
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.DeleteOrg(w, r) })
	if w.Code != http.StatusForbidden {
		t.Errorf("non-owner delete org: got %d, want 403", w.Code)
	}
}

func TestOrgTeamNotFoundIsHidden(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	if _, err := env.orgs.Create("acme", "acme", "", alice.ID); err != nil {
		t.Fatal(err)
	}
	h := NewOrgHandler(env.db, env.cfg, env.repos)

	w := runOrg(t, env, h, alice, http.MethodDelete, map[string]string{"org": "acme", "teamID": "999"}, nil,
		func(h *OrgHandler, w http.ResponseWriter, r *http.Request) { h.DeleteTeam(w, r) })
	if w.Code != http.StatusNotFound {
		t.Errorf("delete unknown team: got %d, want 404", w.Code)
	}
}
