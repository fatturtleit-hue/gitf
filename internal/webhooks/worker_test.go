package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"gitf/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Organization{},
		&models.Webhook{},
		&models.WebhookDelivery{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

type capturedRequest struct {
	body  string
	event string
	sig   string
	ctype string
}

func newCaptureServer(t *testing.T, status int, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		hits.Add(1)
		captured = append(captured, capturedRequest{
			body:  string(body),
			event: r.Header.Get("X-Gitf-Event"),
			sig:   r.Header.Get("X-Gitf-Signature"),
			ctype: r.Header.Get("Content-Type"),
		})
		w.WriteHeader(status)
	}))
}

// Global capture state — webhook tests run sequentially in one package.
var captured []capturedRequest

func resetCapture() { captured = nil }

func TestDeliverSuccessWithSignature(t *testing.T) {
	resetCapture()
	db := newTestDB(t)
	var hits atomic.Int32
	srv := newCaptureServer(t, http.StatusOK, &hits)
	defer srv.Close()

	d := NewDispatcher(db, 1, 5, 1) // maxRetry=1: no retry backoff in tests

	payload := `{"event":"push","branch":"main"}`
	wb := models.Webhook{URL: srv.URL, Secret: "s3cret", ContentType: "json", Events: "push", Active: true}
	d.Enqueue(wb, "push", payload)
	d.Shutdown() // drains the queue; called exactly once

	if hits.Load() != 1 {
		t.Fatalf("server hits = %d, want 1", hits.Load())
	}
	got := captured[0]
	if got.body != payload {
		t.Errorf("body = %q, want %q", got.body, payload)
	}
	if got.event != "push" {
		t.Errorf("X-Gitf-Event = %q", got.event)
	}
	if got.ctype != "application/json" {
		t.Errorf("Content-Type = %q", got.ctype)
	}
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write([]byte(payload))
	wantSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if got.sig != wantSig {
		t.Errorf("signature = %q, want %q", got.sig, wantSig)
	}

	var deliveries []models.WebhookDelivery
	db.Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(deliveries))
	}
	dl := deliveries[0]
	if dl.Status != 200 || dl.Attempt != 1 || dl.DeliveredAt == nil || dl.Error != "" {
		t.Errorf("delivery = %+v", dl)
	}
}

func TestEventFiltering(t *testing.T) {
	resetCapture()
	db := newTestDB(t)
	var hits atomic.Int32
	srv := newCaptureServer(t, http.StatusOK, &hits)
	defer srv.Close()

	d := NewDispatcher(db, 1, 5, 1)

	// Webhook subscribed to issues only — a push must be silently skipped.
	d.Enqueue(models.Webhook{URL: srv.URL, Events: "issues", Active: true}, "push", "{}")
	d.Shutdown() // wait for the worker to (not) deliver

	if hits.Load() != 0 {
		t.Errorf("server should not be hit, got %d", hits.Load())
	}
	var deliveries []models.WebhookDelivery
	db.Find(&deliveries)
	if len(deliveries) != 0 {
		t.Errorf("no delivery rows expected, got %d", len(deliveries))
	}
}

func TestFailedDeliveryRecorded(t *testing.T) {
	resetCapture()
	db := newTestDB(t)
	var hits atomic.Int32
	srv := newCaptureServer(t, http.StatusInternalServerError, &hits)
	defer srv.Close()

	d := NewDispatcher(db, 1, 5, 1)

	d.Enqueue(models.Webhook{URL: srv.URL, Events: "push", Active: true}, "push", "{}")
	d.Shutdown() // wait for the delivery attempt

	var deliveries []models.WebhookDelivery
	db.Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1 (maxRetry=1)", len(deliveries))
	}
	// DeliveredAt means "received an HTTP response" — set even for non-2xx.
	if deliveries[0].Status != 500 {
		t.Errorf("status = %d, want 500", deliveries[0].Status)
	}
	if deliveries[0].DeliveredAt == nil {
		t.Error("DeliveredAt should be set when a response arrived")
	}
}

func TestTransportErrorRecorded(t *testing.T) {
	resetCapture()
	db := newTestDB(t)
	d := NewDispatcher(db, 1, 5, 1)

	// Port 9 (discard) refuses connections — a transport error, not an HTTP status.
	d.Enqueue(models.Webhook{URL: "http://127.0.0.1:9/hook", Events: "push", Active: true}, "push", "{}")
	d.Shutdown()

	var deliveries []models.WebhookDelivery
	db.Find(&deliveries)
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(deliveries))
	}
	if deliveries[0].Status != 0 {
		t.Errorf("status = %d, want 0 (no HTTP response)", deliveries[0].Status)
	}
	if deliveries[0].Error == "" {
		t.Error("transport error should be recorded")
	}
	if deliveries[0].DeliveredAt != nil {
		t.Error("DeliveredAt should be nil without an HTTP response")
	}
}

func TestDispatchWebhookFanOut(t *testing.T) {
	resetCapture()
	db := newTestDB(t)
	var hits atomic.Int32
	srv := newCaptureServer(t, http.StatusOK, &hits)
	defer srv.Close()

	// org-owned repo with one repo-level webhook and one org-level webhook.
	org := models.Organization{Name: "acme"}
	db.Create(&org)
	repo := models.Repository{OwnerType: "org", OwnerID: org.ID, Name: "proj"}
	db.Create(&repo)
	repoHook := models.Webhook{OwnerType: "repo", OwnerID: repo.ID, URL: srv.URL, Events: "push", Active: true}
	orgHook := models.Webhook{OwnerType: "org", OwnerID: org.ID, URL: srv.URL, Events: "push", Active: true}
	db.Create(&repoHook)
	db.Create(&orgHook)

	d := NewDispatcher(db, 1, 5, 1)
	DispatchWebhook(db, d, "repo", repo.ID, "push", `{"x":1}`)
	d.Shutdown()

	// Regression: the org fan-out used to re-dispatch repo webhooks (GORM
	// Find appends to a reused slice).
	if hits.Load() != 2 {
		t.Errorf("server hits = %d, want exactly 2 (repo + org, no duplicates)", hits.Load())
	}

	// nil dispatcher is a safe no-op.
	DispatchWebhook(db, nil, "repo", repo.ID, "push", "{}")
}

func TestPayloadMarshalingRoundTrip(t *testing.T) {
	// The payload is opaque JSON — make sure json is importable and valid
	// payloads survive a marshal/unmarshal round trip.
	raw := map[string]any{"event": "push", "after": "abc"}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back["after"] != "abc" {
		t.Errorf("round trip lost data: %v", back)
	}
}
