package models

import "time"

// Issue represents an issue or pull request in a repository.
// PRs share this table via IsPR=true (Gitea pattern).
type Issue struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	RepoID      uint       `json:"repo_id" gorm:"not null;index"`
	Number      int        `json:"number" gorm:"not null;index"`
	Title       string     `json:"title" gorm:"not null"`
	Body        string     `json:"body"`
	AuthorID    uint       `json:"author_id" gorm:"not null"`
	Author      User       `json:"author" gorm:"constraint:OnDelete:CASCADE;"`
	AssigneeID  *uint      `json:"assignee_id"`
	Assignee    *User      `json:"assignee" gorm:"constraint:OnDelete:SET NULL;"`
	Status      string     `json:"status" gorm:"not null;default:open"` // open, closed
	IsPR        bool       `json:"is_pr" gorm:"default:false"`
	PRID        *uint      `json:"pr_id"`                                    // populated when IsPR
	MilestoneID *uint      `json:"milestone_id"`
	ClosedAt    *time.Time `json:"closed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Issue) TableName() string { return "issues" }

type IssueDTO struct {
	ID              uint    `json:"id"`
	RepoID          uint    `json:"repo_id"`
	Number          int     `json:"number"`
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	AuthorID        uint    `json:"author_id"`
	AuthorUsername  string  `json:"author_username"`
	AssigneeID      *uint   `json:"assignee_id"`
	AssigneeUsername string `json:"assignee_username,omitempty"`
	Status          string  `json:"status"`
	IsPR            bool    `json:"is_pr"`
	PRID            *uint   `json:"pr_id,omitempty"`
	MilestoneID     *uint   `json:"milestone_id,omitempty"`
	ClosedAt        *string `json:"closed_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func (i *Issue) DTO() IssueDTO {
	dto := IssueDTO{
		ID:             i.ID,
		RepoID:         i.RepoID,
		Number:         i.Number,
		Title:          i.Title,
		Body:           i.Body,
		AuthorID:       i.AuthorID,
		AuthorUsername: i.Author.Username,
		AssigneeID:     i.AssigneeID,
		Status:         i.Status,
		IsPR:           i.IsPR,
		PRID:           i.PRID,
		MilestoneID:    i.MilestoneID,
		CreatedAt:      i.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      i.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if i.Assignee != nil {
		dto.AssigneeUsername = i.Assignee.Username
	}
	if i.ClosedAt != nil {
		s := i.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
		dto.ClosedAt = &s
	}
	return dto
}
