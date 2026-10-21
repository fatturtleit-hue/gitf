package issues

import (
	"fmt"
	"time"

	"gitf/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{DB: db} }

// ── Issues ─────────────────────────────────────────────

func (s *Service) Create(repoID uint, authorID uint, title, body string, assigneeID *uint, milestoneID *uint, labelIDs []uint) (*models.Issue, error) {
	// Get next issue number for the repo.
	var maxNum int
	s.DB.Model(&models.Issue{}).Where("repo_id = ?", repoID).
		Select("COALESCE(MAX(number), 0)").Scan(&maxNum)
	issue := models.Issue{
		RepoID:      repoID,
		Number:      maxNum + 1,
		Title:       title,
		Body:        body,
		AuthorID:    authorID,
		AssigneeID:  assigneeID,
		MilestoneID: milestoneID,
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&issue).Error; err != nil {
			return err
		}
		for _, lid := range labelIDs {
			if err := tx.Create(&models.IssueLabel{IssueID: issue.ID, LabelID: lid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.DB.Preload("Author").Preload("Assignee").First(&issue, issue.ID)
	return &issue, nil
}

func (s *Service) GetByNumber(repoID uint, number int) (*models.Issue, error) {
	var issue models.Issue
	if err := s.DB.Preload("Author").Preload("Assignee").
		Where("repo_id = ? AND number = ?", repoID, number).
		First(&issue).Error; err != nil {
		return nil, ErrNotFound
	}
	return &issue, nil
}

func (s *Service) List(repoID uint, status, label, milestone, assignee, search string, limit, offset int) ([]models.Issue, int, error) {
	query := s.DB.Preload("Author").Preload("Assignee").Where("repo_id = ?", repoID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR body LIKE ?", like, like)
	}
	if label != "" {
		query = query.Joins("JOIN issue_labels il ON il.issue_id = issues.id").
			Joins("JOIN labels l ON l.id = il.label_id").
			Where("l.name = ?", label)
	}
	if milestone != "" {
		query = query.Joins("JOIN milestones m ON m.id = issues.milestone_id").
			Where("m.title = ?", milestone)
	}
	if assignee != "" {
		query = query.Joins("JOIN users u ON u.id = issues.assignee_id").
			Where("u.username = ?", assignee)
	}

	var total int64
	if err := query.Model(&models.Issue{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var issues []models.Issue
	if err := query.Order("issues.updated_at DESC").Limit(limit).Offset(offset).Find(&issues).Error; err != nil {
		return nil, 0, err
	}
	return issues, int(total), nil
}

func (s *Service) Update(issue *models.Issue) error {
	return s.DB.Omit(clause.Associations).Save(issue).Error
}

func (s *Service) Close(issue *models.Issue, userID uint) error {
	now := time.Now()
	issue.Status = "closed"
	issue.ClosedAt = &now
	return s.DB.Omit(clause.Associations).Save(issue).Error
}

func (s *Service) Reopen(issue *models.Issue) error {
	issue.Status = "open"
	issue.ClosedAt = nil
	return s.DB.Omit(clause.Associations).Save(issue).Error
}

// ── Comments ───────────────────────────────────────────

func (s *Service) AddComment(repoID, issueID, authorID uint, body string) (*models.Comment, error) {
	c := models.Comment{RepoID: repoID, IssueID: issueID, AuthorID: authorID, Body: body}
	if err := s.DB.Create(&c).Error; err != nil {
		return nil, err
	}
	s.DB.Preload("Author").First(&c, c.ID)
	return &c, nil
}

func (s *Service) ListComments(issueID uint) ([]models.Comment, error) {
	var comments []models.Comment
	if err := s.DB.Preload("Author").Where("issue_id = ?", issueID).
		Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

// ── Labels ─────────────────────────────────────────────

func (s *Service) CreateLabel(repoID uint, name, color string) (*models.Label, error) {
	l := models.Label{RepoID: repoID, Name: name, Color: color}
	if err := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&l).Error; err != nil {
		return nil, err
	}
	if l.ID == 0 {
		s.DB.Where("repo_id = ? AND name = ?", repoID, name).First(&l)
	}
	return &l, nil
}

func (s *Service) ListLabels(repoID uint) ([]models.Label, error) {
	var labels []models.Label
	if err := s.DB.Where("repo_id = ?", repoID).Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

func (s *Service) DeleteLabel(labelID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", labelID).Delete(&models.IssueLabel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Label{}, labelID).Error
	})
}

func (s *Service) AddIssueLabel(issueID, labelID uint) error {
	il := models.IssueLabel{IssueID: issueID, LabelID: labelID}
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&il).Error
}

func (s *Service) RemoveIssueLabel(issueID, labelID uint) error {
	return s.DB.Where("issue_id = ? AND label_id = ?", issueID, labelID).Delete(&models.IssueLabel{}).Error
}

// ── Milestones ─────────────────────────────────────────

func (s *Service) CreateMilestone(repoID uint, title, description string, dueDate *string) (*models.Milestone, error) {
	m := models.Milestone{RepoID: repoID, Title: title, Description: description}
	if dueDate != nil && *dueDate != "" {
		t, err := time.Parse("2006-01-02", *dueDate)
		if err == nil {
			m.DueDate = &t
		}
	}
	if err := s.DB.Create(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) ListMilestones(repoID uint) ([]models.Milestone, error) {
	var milestones []models.Milestone
	if err := s.DB.Where("repo_id = ?", repoID).Find(&milestones).Error; err != nil {
		return nil, err
	}
	return milestones, nil
}

func (s *Service) CloseMilestone(milestone *models.Milestone) error {
	milestone.Status = "closed"
	return s.DB.Omit(clause.Associations).Save(milestone).Error
}

// ── Kanban ─────────────────────────────────────────────

func (s *Service) GetColumns(repoID uint) ([]models.IssueColumn, error) {
	// Ensure default columns exist.
	s.CreateDefaultColumns(repoID)
	var cols []models.IssueColumn
	if err := s.DB.Where("repo_id = ?", repoID).Order("position").Find(&cols).Error; err != nil {
		return nil, err
	}
	return cols, nil
}

func (s *Service) CreateDefaultColumns(repoID uint) error {
	defaults := []models.IssueColumn{
		{RepoID: repoID, Name: "待办", Position: 0},
		{RepoID: repoID, Name: "进行中", Position: 1},
		{RepoID: repoID, Name: "已完成", Position: 2},
	}
	for _, col := range defaults {
		s.DB.Clauses(clause.OnConflict{DoNothing: true}).Where("repo_id = ? AND name = ?", repoID, col.Name).Create(&col)
	}
	return nil
}

func (s *Service) MoveIssue(issueID, columnID uint, position int) error {
	item := models.IssueColumnItem{
		IssueID:  issueID,
		ColumnID: columnID,
		Position: position,
	}
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "issue_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"column_id", "position"}),
	}).Create(&item).Error
}

func (s *Service) GetColumnIssues(columnID uint) ([]models.Issue, error) {
	var items []models.IssueColumnItem
	if err := s.DB.Where("column_id = ?", columnID).Order("position").Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []models.Issue{}, nil
	}
	ids := make([]uint, len(items))
	for i, item := range items {
		ids[i] = item.IssueID
	}
	var issues []models.Issue
	if err := s.DB.Preload("Author").Preload("Assignee").Where("id IN ?", ids).Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

// Ensure fmt is used (for future error formatting).
var _ = fmt.Sprintf("")
