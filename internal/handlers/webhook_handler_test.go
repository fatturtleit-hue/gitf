package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitf/internal/models"
	"gitf/internal/webhooks"
)

// webhookRequest runs a webhook-scoped handler call.
func webhookRequest(t *testing.T, env *htEnv, h *WebhookHandler, user *models.User, params map[string]string, body any, call func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
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

func TestWebhookAdminOnly(t *testing.T) {
	env := newHandlerEnv(t)
	alice := env.user(t, "alice")
	bob := env.user(t, "bob")
	env.seedRepo(t, "proj", false)
	h := NewWebhookHandler(env.db, env.repos, webhooks.NewDispatcher(env.db, 1, 5, 1))

	body := map[string]string{"url": "http://127.0.0.1:9/hook", "events": "push"}

	// REGRESSION (security): non-admin must not manage webhooks.
	w := webhookRequest(t, env, h, bob, nil, body, h.CreateWebhook)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin create webhook: got %d, want 403", w.Code)
	}

	w = webhookRequest(t, env, h, alice, nil, body, h.CreateWebhook)
	if w.Code != http.StatusCreated {
		t.Fatalf("create webhook: got %d: %s", w.Code, w.Body.String())
	}

	w = webhookRequest(t, env, h, alice, nil, nil, h.ListWebhooks)
	if w.Code != http.StatusOK {
		t.Errorf("list webhooks: got %d", w.Code)
	}

	w = webhookRequest(t, env, h, alice, map[string]string{"hookID": "1"}, nil, h.DeleteWebhook)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete webhook: got %d", w.Code)
	}
}
