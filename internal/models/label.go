package models

// Label represents a categorisation tag for issues and pull requests.
type Label struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	RepoID uint  `json:"repo_id" gorm:"not null;index;uniqueIndex:idx_repo_name"`
	Name  string `json:"name" gorm:"not null;uniqueIndex:idx_repo_name"`
	Color string `json:"color" gorm:"not null;default:'#6b7280'"`
}

func (Label) TableName() string { return "labels" }

type LabelDTO struct {
	ID    uint   `json:"id"`
	RepoID uint  `json:"repo_id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (l *Label) DTO() LabelDTO {
	return LabelDTO{ID: l.ID, RepoID: l.RepoID, Name: l.Name, Color: l.Color}
}

// IssueLabel is the join table between issues and labels.
type IssueLabel struct {
	IssueID uint `json:"issue_id" gorm:"primaryKey"`
	LabelID uint `json:"label_id" gorm:"primaryKey"`
}

func (IssueLabel) TableName() string { return "issue_labels" }
