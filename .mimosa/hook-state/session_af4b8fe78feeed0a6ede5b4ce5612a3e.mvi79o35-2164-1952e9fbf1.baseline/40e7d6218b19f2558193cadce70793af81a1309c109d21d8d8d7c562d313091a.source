package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitf/internal/models"

	"gorm.io/gorm"
)

// DeliveryJob is a webhook delivery request.
type DeliveryJob struct {
	Webhook models.Webhook
	Event   string
	Payload string
}

// Dispatcher manages the webhook delivery worker pool.
type Dispatcher struct {
	db       *gorm.DB
	queue    chan DeliveryJob
	workers  int
	timeout  time.Duration
	maxRetry int
	wg       sync.WaitGroup
}

// NewDispatcher creates a webhook delivery dispatcher and starts workers.
func NewDispatcher(db *gorm.DB, workerCount, timeoutSec, maxRetry int) *Dispatcher {
	d := &Dispatcher{
		db:       db,
		queue:    make(chan DeliveryJob, 1000),
		workers:  workerCount,
		timeout:  time.Duration(timeoutSec) * time.Second,
		maxRetry: maxRetry,
	}
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
	return d
}

// Enqueue queues a webhook delivery. Non-blocking — drops if queue is full.
func (d *Dispatcher) Enqueue(webhook models.Webhook, event, payload string) {
	job := DeliveryJob{Webhook: webhook, Event: event, Payload: payload}
	select {
	case d.queue <- job:
	default:
		slog.Warn("webhook queue full, dropping delivery", "webhook_id", webhook.ID)
	}
}

// Shutdown drains the queue and stops workers.
func (d *Dispatcher) Shutdown() {
	close(d.queue)
	d.wg.Wait()
}

// QueueCapacity returns the current queue length.
func (d *Dispatcher) QueueCapacity() int { return len(d.queue) }

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for job := range d.queue {
		d.deliver(job)
	}
}

func (d *Dispatcher) deliver(job DeliveryJob) {
	// Filter events: only deliver if the webhook subscribes to this event.
	events := strings.Fields(job.Webhook.Events)
	matched := false
	for _, e := range events {
		if e == job.Event {
			matched = true
			break
		}
	}
	if !matched {
		return
	}

	client := &http.Client{Timeout: d.timeout}

	for attempt := 1; attempt <= d.maxRetry; attempt++ {
		status, errStr := d.send(client, job)
		delivery := models.WebhookDelivery{
			WebhookID: job.Webhook.ID,
			Event:     job.Event,
			Payload:   job.Payload,
			Status:    status,
			Attempt:   attempt,
			Error:     errStr,
		}
		now := time.Now()
		if status > 0 {
			delivery.DeliveredAt = &now
		}
		d.db.Create(&delivery)

		if status >= 200 && status < 300 {
			return
		}
		if attempt < d.maxRetry {
			backoff := time.Duration(attempt) * time.Second * 10
			time.Sleep(backoff)
		}
	}
}

func (d *Dispatcher) send(client *http.Client, job DeliveryJob) (status int, errStr string) {
	payload := []byte(job.Payload)
	req, err := http.NewRequest(http.MethodPost, job.Webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, err.Error()
	}

	contentType := job.Webhook.ContentType
	if contentType == "" {
		contentType = "json"
	}
	req.Header.Set("Content-Type", "application/"+contentType)
	req.Header.Set("X-Gitf-Event", job.Event)

	// HMAC-SHA256 signing.
	if job.Webhook.Secret != "" {
		mac := hmac.New(sha256.New, []byte(job.Webhook.Secret))
		mac.Write(payload)
		req.Header.Set("X-Gitf-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	return resp.StatusCode, ""
}

// DispatchWebhook is a convenience function called from handlers.
// It loads matching webhooks for the given owner and fires deliveries.
func DispatchWebhook(db *gorm.DB, dispatcher *Dispatcher, ownerType string, ownerID uint, event, payloadJSON string) {
	if dispatcher == nil {
		return
	}
	// Also trigger repo-level and org-level webhooks.
	var webhooks []models.Webhook
	db.Where("owner_type = ? AND owner_id = ? AND active = ?", ownerType, ownerID, true).Find(&webhooks)
	for _, w := range webhooks {
		dispatcher.Enqueue(w, event, payloadJSON)
	}
	// If owner_type is repo, also check org ownership.
	if ownerType == "repo" {
		var repo models.Repository
		if err := db.First(&repo, ownerID).Error; err == nil && repo.OwnerType == "org" {
			db.Where("owner_type = ? AND owner_id = ? AND active = ?", "org", repo.OwnerID, true).Find(&webhooks)
			for _, w := range webhooks {
				dispatcher.Enqueue(w, event, payloadJSON)
			}
		}
	}
}

// Ensure fmt and json are used.
var _ = fmt.Sprintf("")
var _ = json.Marshal
