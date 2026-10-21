package models

import "time"

// Webhook represents an outbound webhook for a user, org, or repo.
type Webhook struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OwnerType   string    `json:"owner_type" gorm:"not null;index"` // user, org, repo
	OwnerID     uint      `json:"owner_id" gorm:"not null;index"`
	URL         string    `json:"url" gorm:"not null"`
	Secret      string    `json:"secret"`
	ContentType string    `json:"content_type" gorm:"default:json"` // json, form
	Active      bool      `json:"active" gorm:"default:true"`
	Events      string    `json:"events" gorm:"not null"` // space-separated: push, pull_request, issues, release, package
	CreatedAt   time.Time `json:"created_at"`
}

func (Webhook) TableName() string { return "webhooks" }

type WebhookDTO struct {
	ID          uint   `json:"id"`
	OwnerType   string `json:"owner_type"`
	OwnerID     uint   `json:"owner_id"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Active      bool   `json:"active"`
	Events      string `json:"events"`
	CreatedAt   string `json:"created_at"`
}

func (w *Webhook) DTO() WebhookDTO {
	return WebhookDTO{
		ID: w.ID, OwnerType: w.OwnerType, OwnerID: w.OwnerID,
		URL: w.URL, ContentType: w.ContentType, Active: w.Active,
		Events: w.Events, CreatedAt: w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// WebhookDelivery represents a single delivery attempt.
type WebhookDelivery struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	WebhookID   uint       `json:"webhook_id" gorm:"not null;index"`
	Event       string     `json:"event" gorm:"not null"`
	Payload     string     `json:"payload" gorm:"type:text"`
	Status      int        `json:"status"`
	Attempt     int        `json:"attempt" gorm:"default:1"`
	DeliveredAt *time.Time `json:"delivered_at"`
	Error       string     `json:"error"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }
