package models

import "time"

// PullRequest represents a pull request linked 1:1 to an Issue (IsPR=true).
type PullRequest struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	RepoID          uint       `json:"repo_id" gorm:"not null;index"`
	IssueID         uint       `json:"issue_id" gorm:"uniqueIndex"` // 1:1 with Issue where IsPR=true
	HeadRepoID      uint       `json:"head_repo_id" gorm:"not null"`
	HeadBranch      string     `json:"head_branch" gorm:"not null"`
	BaseBranch      string     `json:"base_branch" gorm:"not null"`
	Status          string     `json:"status" gorm:"not null;default:open"` // open, closed, merged
	MergeCommitSHA  *string    `json:"merge_commit_sha"`
	MergeStrategy   *string    `json:"merge_strategy"` // merge, squash, rebase
	MergedByID      *uint      `json:"merged_by_id"`
	MergedBy        *User      `json:"merged_by" gorm:"constraint:OnDelete:SET NULL;"`
	MergedAt        *time.Time `json:"merged_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (PullRequest) TableName() string { return "pull_requests" }

type PullRequestDTO struct {
	ID             uint    `json:"id"`
	RepoID         uint    `json:"repo_id"`
	IssueID        uint    `json:"issue_id"`
	HeadRepoID     uint    `json:"head_repo_id"`
	HeadBranch     string  `json:"head_branch"`
	BaseBranch     string  `json:"base_branch"`
	Status         string  `json:"status"`
	MergeCommitSHA *string `json:"merge_commit_sha,omitempty"`
	MergeStrategy  *string `json:"merge_strategy,omitempty"`
	MergedByID     *uint   `json:"merged_by_id,omitempty"`
	MergedAt       *string `json:"merged_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

func (pr *PullRequest) DTO() PullRequestDTO {
	dto := PullRequestDTO{
		ID:             pr.ID,
		RepoID:         pr.RepoID,
		IssueID:        pr.IssueID,
		HeadRepoID:     pr.HeadRepoID,
		HeadBranch:     pr.HeadBranch,
		BaseBranch:     pr.BaseBranch,
		Status:         pr.Status,
		MergeCommitSHA: pr.MergeCommitSHA,
		MergeStrategy:  pr.MergeStrategy,
		MergedByID:     pr.MergedByID,
		CreatedAt:      pr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if pr.MergedAt != nil {
		s := pr.MergedAt.Format("2006-01-02T15:04:05Z07:00")
		dto.MergedAt = &s
	}
	return dto
}

// Review represents a review on a pull request.
type Review struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	PRID        uint      `json:"pr_id" gorm:"not null;index"`
	ReviewerID  uint      `json:"reviewer_id" gorm:"not null"`
	Reviewer    User      `json:"reviewer" gorm:"constraint:OnDelete:CASCADE;"`
	Body        string    `json:"body"`
	Status      string    `json:"status" gorm:"not null;default:commented"` // requested, approved, changes_requested, commented
	CommitSHA   string    `json:"commit_sha"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Review) TableName() string { return "reviews" }

type ReviewDTO struct {
	ID               uint   `json:"id"`
	PRID             uint   `json:"pr_id"`
	ReviewerID       uint   `json:"reviewer_id"`
	ReviewerUsername string `json:"reviewer_username"`
	Body             string `json:"body"`
	Status           string `json:"status"`
	CommitSHA        string `json:"commit_sha"`
	CreatedAt        string `json:"created_at"`
}

func (r *Review) DTO() ReviewDTO {
	return ReviewDTO{
		ID:               r.ID,
		PRID:             r.PRID,
		ReviewerID:       r.ReviewerID,
		ReviewerUsername: r.Reviewer.Username,
		Body:             r.Body,
		Status:           r.Status,
		CommitSHA:        r.CommitSHA,
		CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
