package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitf/internal/models"
)

// releaseRequest runs a release-scoped handler call.
func releaseRequest(t *testing.T, env *htEnv, h *ReleaseHandler, user *models.User, params map[string]string, body any, call func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
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

func TestReleaseLifecycle(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	bob := env.user(t, "bob")
	env.seedRepo(t, "proj", false)
	h := NewReleaseHandler(env.db, env.cfg, env.repos)
	_ = bob

	w := releaseRequest(t, env, h, alice, nil,
		map[string]string{"tag_name": "v1.0.0", "name": "first", "body": "notes"}, h.CreateRelease)
	if w.Code != http.StatusCreated {
		t.Fatalf("create release: got %d: %s", w.Code, w.Body.String())
	}

	w = releaseRequest(t, env, h, alice, nil, nil, h.ListReleases)
	if w.Code != http.StatusOK {
		t.Fatalf("list releases: got %d", w.Code)
	}

	w = releaseRequest(t, env, h, alice, map[string]string{"releaseID": "1"}, nil, h.GetRelease)
	if w.Code != http.StatusOK {
		t.Fatalf("get release: got %d", w.Code)
	}

	// Upload an asset via multipart form (10MB cap, field name "file").
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "asset.bin")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("binary-bytes"))
	mw.Close()
	req := chiContext(t, asUser(httptest.NewRequest(http.MethodPost, "/api/v1", &buf), alice),
		map[string]string{"owner": "alice", "name": "proj", "releaseID": "1"})
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w = httptest.NewRecorder()
	h.UploadAsset(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("upload asset: got %d: %s", w.Code, w.Body.String())
	}

	w = releaseRequest(t, env, h, alice, map[string]string{"releaseID": "1"}, nil, h.DeleteRelease)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete release: got %d", w.Code)
	}
}

func TestReleaseNonAdminCannotDelete(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	bob := env.user(t, "bob")
	env.seedRepo(t, "proj", false)
	h := NewReleaseHandler(env.db, env.cfg, env.repos)

	releaseRequest(t, env, h, alice, nil,
		map[string]string{"tag_name": "v1", "name": "x"}, h.CreateRelease)

	// bob has no write access to alice's repo → 403.
	w := releaseRequest(t, env, h, bob, map[string]string{"releaseID": "1"}, nil, h.DeleteRelease)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin delete release: got %d, want 403", w.Code)
	}
}
