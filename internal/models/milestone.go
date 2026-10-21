package models

import "time"

// Milestone represents a project milestone for a repository.
type Milestone struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	RepoID    uint       `json:"repo_id" gorm:"not null;index;uniqueIndex:idx_repo_title"`
	Title     string     `json:"title" gorm:"not null;uniqueIndex:idx_repo_title"`
	Description string   `json:"description"`
	DueDate   *time.Time `json:"due_date"`
	Status    string     `json:"status" gorm:"not null;default:open"` // open, closed
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Milestone) TableName() string { return "milestones" }

type MilestoneDTO struct {
	ID          uint    `json:"id"`
	RepoID      uint    `json:"repo_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueDate     *string `json:"due_date,omitempty"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

func (m *Milestone) DTO() MilestoneDTO {
	dto := MilestoneDTO{
		ID:          m.ID,
		RepoID:      m.RepoID,
		Title:       m.Title,
		Description: m.Description,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if m.DueDate != nil {
		s := m.DueDate.Format("2006-01-02T15:04:05Z07:00")
		dto.DueDate = &s
	}
	return dto
}
