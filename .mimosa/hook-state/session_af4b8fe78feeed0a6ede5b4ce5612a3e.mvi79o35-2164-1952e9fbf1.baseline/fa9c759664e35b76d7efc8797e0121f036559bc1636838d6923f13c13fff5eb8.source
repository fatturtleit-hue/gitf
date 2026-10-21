package handlers

import (
	"net/http"

	"gitf/internal/models"
	"gitf/internal/repo"
	"gitf/internal/webhooks"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	Repos      repo.RepositoryStore
	DB         *gorm.DB
	Dispatcher *webhooks.Dispatcher
}

func NewWebhookHandler(db *gorm.DB, repos repo.RepositoryStore, dispatcher *webhooks.Dispatcher) *WebhookHandler {
	return &WebhookHandler{DB: db, Repos: repos, Dispatcher: dispatcher}
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	var req struct {
		URL         string `json:"url"`
		Secret      string `json:"secret"`
		ContentType string `json:"content_type"`
		Events      string `json:"events"`
		Active      *bool  `json:"active"`
	}
	if !decodeJSON(w, r, &req) { return }
	if req.ContentType == "" { req.ContentType = "json" }
	active := true
	if req.Active != nil { active = *req.Active }

	wb := models.Webhook{
		OwnerType:   "repo",
		OwnerID:     repository.ID,
		URL:         req.URL,
		Secret:      req.Secret,
		ContentType: req.ContentType,
		Active:      active,
		Events:      req.Events,
	}
	if err := h.DB.Create(&wb).Error; err != nil {
		writeServerError(w, r, "create webhook", err)
		return
	}
	writeJSON(w, r, http.StatusCreated, wb.DTO())
}

func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	var hooks []models.Webhook
	h.DB.Where("owner_type = ? AND owner_id = ?", "repo", repository.ID).Find(&hooks)
	items := make([]models.WebhookDTO, len(hooks))
	for i, w := range hooks {
		items[i] = w.DTO()
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": items})
}

func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	id := atoiOrZero(chi.URLParam(r, "hookID"))
	h.DB.Where("owner_type = ? AND owner_id = ? AND id = ?", "repo", repository.ID, id).Delete(&models.Webhook{})
	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	id := atoiOrZero(chi.URLParam(r, "hookID"))
	var deliveries []models.WebhookDelivery
	h.DB.Where("webhook_id = ?", id).Order("created_at DESC").Limit(50).Find(&deliveries)
	writeJSON(w, r, http.StatusOK, map[string]any{"items": deliveries})
}

func (h *WebhookHandler) Redeliver(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok { return }
	repository, ok := repoFromPRHandler(h.Repos, w, r)
	if !ok { return }
	if !h.Repos.CanAdmin(repository, user) {
		writeAPIError(w, r, http.StatusForbidden, ErrCodeForbidden, "access denied")
		return
	}
	id := atoiOrZero(chi.URLParam(r, "hookID"))
	deliveryID := atoiOrZero(chi.URLParam(r, "deliveryID"))
	var delivery models.WebhookDelivery
	if err := h.DB.Where("webhook_id = ? AND id = ?", id, deliveryID).First(&delivery).Error; err != nil {
		writeAPIError(w, r, http.StatusNotFound, ErrCodeNotFound, "delivery not found")
		return
	}
	// Re-enqueue.
	var wb models.Webhook
	if err := h.DB.First(&wb, id).Error; err == nil {
		h.Dispatcher.Enqueue(wb, delivery.Event, delivery.Payload)
	}
	w.WriteHeader(http.StatusAccepted)
}
