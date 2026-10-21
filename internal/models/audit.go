
package models

import (
	"time"

	"gorm.io/gorm"
)

// AuditLog represents an audit log entry for security compliance.
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	Action    string    `json:"action" gorm:"not null;index"`
	Resource  string    `json:"resource" gorm:"not null"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName overrides GORM default pluralization.
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditStore defines the interface for audit log operations.
type AuditStore interface {
	Log(userID uint, action, resource string) error
}

// GormAuditStore implements AuditStore using GORM.
type GormAuditStore struct {
	DB *gorm.DB
}

// NewGormAuditStore creates a new GORM-based audit store.
func NewGormAuditStore(db *gorm.DB) AuditStore {
	return &GormAuditStore{DB: db}
}

// Log creates an audit log entry.
func (s *GormAuditStore) Log(userID uint, action, resource string) error {
	entry := AuditLog{
		UserID:   userID,
		Action:   action,
		Resource: resource,
	}
	return s.DB.Create(&entry).Error
}
