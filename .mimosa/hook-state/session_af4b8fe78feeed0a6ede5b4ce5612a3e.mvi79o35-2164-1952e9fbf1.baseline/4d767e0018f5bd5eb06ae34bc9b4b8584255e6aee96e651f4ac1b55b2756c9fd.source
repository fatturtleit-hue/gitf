package models

import "time"

// Collaborator represents a user's access to a repository.
type Collaborator struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	RepoID     uint      `json:"repo_id" gorm:"not null;index;uniqueIndex:idx_repo_user"`
	UserID     uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_repo_user"`
	User       User      `json:"user" gorm:"constraint:OnDelete:CASCADE;"`
	Permission string    `json:"permission" gorm:"not null;default:read"` // read, write, admin
	QuotaBytes int64     `json:"quota_bytes" gorm:"default:0"`           // 0 = inherit repo default
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Collaborator) TableName() string {
	return "collaborators"
}

type CollaboratorDTO struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Permission string `json:"permission"`
	QuotaBytes int64  `json:"quota_bytes"`
	CreatedAt  string `json:"created_at"`
}

func (c *Collaborator) DTO() CollaboratorDTO {
	return CollaboratorDTO{
		ID:         c.ID,
		UserID:     c.UserID,
		Username:   c.User.Username,
		Email:      c.User.Email,
		Permission: c.Permission,
		QuotaBytes: c.QuotaBytes,
		CreatedAt:  c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
