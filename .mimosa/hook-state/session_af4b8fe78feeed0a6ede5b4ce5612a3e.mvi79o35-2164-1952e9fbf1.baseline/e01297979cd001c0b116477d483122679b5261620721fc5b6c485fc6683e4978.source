package models

import "time"

// Comment represents a comment on an issue or pull request.
// Line-comment fields (Path, Line, CommitID, Side) are nil for general issue comments
// and populated for PR line comments (Phase 3).
type Comment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RepoID    uint      `json:"repo_id" gorm:"not null;index"`
	IssueID   uint      `json:"issue_id" gorm:"not null;index"`
	AuthorID  uint      `json:"author_id" gorm:"not null"`
	Author    User      `json:"author" gorm:"constraint:OnDelete:CASCADE;"`
	Body      string    `json:"body" gorm:"type:text"`
	// Line-comment fields for PR reviews (Phase 3).
	Path     *string `json:"path,omitempty"`
	Line     *int    `json:"line,omitempty"`
	CommitID *string `json:"commit_id,omitempty"`
	Side     *string `json:"side,omitempty"` // "left" or "right"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Comment) TableName() string { return "comments" }

type CommentDTO struct {
	ID             uint    `json:"id"`
	IssueID        uint    `json:"issue_id"`
	AuthorID       uint    `json:"author_id"`
	AuthorUsername string  `json:"author_username"`
	Body           string  `json:"body"`
	Path           *string `json:"path,omitempty"`
	Line           *int    `json:"line,omitempty"`
	CommitID       *string `json:"commit_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

func (c *Comment) DTO() CommentDTO {
	return CommentDTO{
		ID:             c.ID,
		IssueID:        c.IssueID,
		AuthorID:       c.AuthorID,
		AuthorUsername: c.Author.Username,
		Body:           c.Body,
		Path:           c.Path,
		Line:           c.Line,
		CommitID:       c.CommitID,
		CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
