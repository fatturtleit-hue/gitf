package models

import "time"

// PersonalAccessToken represents an API token for authenticating with the API or git.
type PersonalAccessToken struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"not null;index"`
	Name       string     `json:"name" gorm:"not null"`
	TokenHash  string     `json:"-" gorm:"uniqueIndex;not null"`
	Prefix     string     `json:"prefix" gorm:"not null"` // first 8 chars for display
	Scopes     string     `json:"scopes" gorm:"not null;default:'repo'"` // space-separated: repo, repo:write, admin, package:write, workflow, read:user
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (PersonalAccessToken) TableName() string { return "personal_access_tokens" }

type PATDTO struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	Prefix    string  `json:"prefix"`
	Scopes    string  `json:"scopes"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func (p *PersonalAccessToken) DTO() PATDTO {
	dto := PATDTO{
		ID:        p.ID,
		Name:      p.Name,
		Prefix:    p.Prefix,
		Scopes:    p.Scopes,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if p.ExpiresAt != nil {
		s := p.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		dto.ExpiresAt = &s
	}
	return dto
}
