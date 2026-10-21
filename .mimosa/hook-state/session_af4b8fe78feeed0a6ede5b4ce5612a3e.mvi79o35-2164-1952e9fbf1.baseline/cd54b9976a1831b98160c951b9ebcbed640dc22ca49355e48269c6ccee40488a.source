package issues

import (
	"errors"

	"gitf/internal/models"
)

var (
	ErrNotFound  = errors.New("issue not found")
	ErrConflict  = errors.New("issue already exists")
)

// Store abstracts issue tracking persistence.
type Store interface {
	Create(repoID uint, authorID uint, title, body string, assigneeID *uint, milestoneID *uint, labelIDs []uint) (*models.Issue, error)
	GetByNumber(repoID uint, number int) (*models.Issue, error)
	List(repoID uint, status string, label string, milestone string, assignee string, search string, limit, offset int) ([]models.Issue, int, error)
	Update(issue *models.Issue) error
	Close(issue *models.Issue, userID uint) error
	Reopen(issue *models.Issue) error

	// Comments
	AddComment(repoID, issueID, authorID uint, body string) (*models.Comment, error)
	ListComments(issueID uint) ([]models.Comment, error)

	// Labels
	CreateLabel(repoID uint, name, color string) (*models.Label, error)
	ListLabels(repoID uint) ([]models.Label, error)
	DeleteLabel(labelID uint) error
	AddIssueLabel(issueID, labelID uint) error
	RemoveIssueLabel(issueID, labelID uint) error

	// Milestones
	CreateMilestone(repoID uint, title, description string, dueDate *string) (*models.Milestone, error)
	ListMilestones(repoID uint) ([]models.Milestone, error)
	CloseMilestone(milestone *models.Milestone) error

	// Kanban
	GetColumns(repoID uint) ([]models.IssueColumn, error)
	CreateDefaultColumns(repoID uint) error
	MoveIssue(issueID, columnID uint, position int) error
	GetColumnIssues(columnID uint) ([]models.Issue, error)
}

var _ Store = (*Service)(nil)
