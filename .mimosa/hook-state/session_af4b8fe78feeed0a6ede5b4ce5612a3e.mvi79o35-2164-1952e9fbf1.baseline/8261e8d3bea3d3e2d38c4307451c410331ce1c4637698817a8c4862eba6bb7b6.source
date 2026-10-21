package models

// IssueColumn defines a column on a kanban board.
type IssueColumn struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	RepoID   uint   `json:"repo_id" gorm:"not null;index"`
	Name     string `json:"name" gorm:"not null"`
	Position int    `json:"position" gorm:"not null;default:0"`
}

func (IssueColumn) TableName() string { return "issue_columns" }

type IssueColumnDTO struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

func (c *IssueColumn) DTO() IssueColumnDTO {
	return IssueColumnDTO{ID: c.ID, Name: c.Name, Position: c.Position}
}

// IssueColumnItem places an issue in a kanban column with a position.
type IssueColumnItem struct {
	IssueID  uint `json:"issue_id" gorm:"uniqueIndex"`
	ColumnID uint `json:"column_id" gorm:"index"`
	Position int  `json:"position" gorm:"default:0"`
}

func (IssueColumnItem) TableName() string { return "issue_column_items" }
