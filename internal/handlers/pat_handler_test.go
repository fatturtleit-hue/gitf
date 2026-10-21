package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestPATLifecycle(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	h := NewPATHandler(env.db)

	// Create → 201 with one-time plaintext token and the record's ID.
	req := chiContext(t, asUser(newTestRequest(t, http.MethodPost, "/user/tokens",
		map[string]string{"name": "ci", "scopes": "repo"}), alice), nil)
	w := httptest.NewRecorder()
	h.CreateToken(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create token: got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Token string `json:"token"`
		Info  struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Token string `json:"-"` // hashed — must never serialize
		} `json:"info"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Token, "gitf_") {
		t.Errorf("plaintext token = %q, want gitf_ prefix", created.Token)
	}
	if created.Info.ID == 0 {
		t.Fatal("info.id missing from create response")
	}

	// List shows the record but never the plaintext token.
	req = chiContext(t, asUser(newTestRequest(t, http.MethodGet, "/user/tokens", nil), alice), nil)
	w = httptest.NewRecorder()
	h.ListTokens(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("list tokens: got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), created.Token) {
		t.Error("plaintext token must not appear in listings")
	}

	// Delete → 204; deleting again → 404.
	req = chiContext(t, asUser(newTestRequest(t, http.MethodDelete, "/user/tokens/"+strconv.FormatUint(uint64(created.Info.ID), 10), nil), alice),
		map[string]string{"tokenID": strconv.FormatUint(uint64(created.Info.ID), 10)})
	w = httptest.NewRecorder()
	h.DeleteToken(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete token: got %d", w.Code)
	}
	req = chiContext(t, asUser(newTestRequest(t, http.MethodDelete, "/user/tokens/"+strconv.FormatUint(uint64(created.Info.ID), 10), nil), alice),
		map[string]string{"tokenID": strconv.FormatUint(uint64(created.Info.ID), 10)})
	w = httptest.NewRecorder()
	h.DeleteToken(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("double delete: got %d, want 404", w.Code)
	}
}

func TestPATNameRequired(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	h := NewPATHandler(env.db)

	req := chiContext(t, asUser(newTestRequest(t, http.MethodPost, "/user/tokens",
		map[string]string{"scopes": "repo"}), alice), nil)
	w := httptest.NewRecorder()
	h.CreateToken(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("token without name: got %d, want 400", w.Code)
	}
}
