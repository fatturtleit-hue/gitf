package models

import "time"

// Organization represents a group that can own repositories.
type Organization struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Organization) TableName() string { return "organizations" }

type OrganizationDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func (o *Organization) DTO() OrganizationDTO {
	return OrganizationDTO{
		ID:          o.ID,
		Name:        o.Name,
		DisplayName: o.DisplayName,
		Description: o.Description,
		CreatedAt:   o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	OrgID     uint      `json:"org_id" gorm:"not null;index;uniqueIndex:idx_org_user"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_org_user"`
	User      User      `json:"user" gorm:"constraint:OnDelete:CASCADE;"`
	Role      string    `json:"role" gorm:"not null;default:member"` // owner, admin, member
	CreatedAt time.Time `json:"created_at"`
}

func (OrgMember) TableName() string { return "org_members" }

type OrgMemberDTO struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (m *OrgMember) DTO() OrgMemberDTO {
	return OrgMemberDTO{
		ID:        m.ID,
		UserID:    m.UserID,
		Username:  m.User.Username,
		Email:     m.User.Email,
		Role:      m.Role,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Team represents a named group within an organization.
type Team struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OrgID       uint      `json:"org_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex:idx_org_team"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Team) TableName() string { return "teams" }

type TeamDTO struct {
	ID          uint   `json:"id"`
	OrgID       uint   `json:"org_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func (t *Team) DTO() TeamDTO {
	return TeamDTO{
		ID:          t.ID,
		OrgID:       t.OrgID,
		Name:        t.Name,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	TeamID uint `json:"team_id" gorm:"not null;uniqueIndex:idx_team_user"`
	UserID uint `json:"user_id" gorm:"not null;uniqueIndex:idx_team_user"`
	User   User `json:"user" gorm:"constraint:OnDelete:CASCADE;"`
}

func (TeamMember) TableName() string { return "team_members" }

type TeamMemberDTO struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

func (m *TeamMember) DTO() TeamMemberDTO {
	return TeamMemberDTO{
		ID:       m.ID,
		UserID:   m.UserID,
		Username: m.User.Username,
	}
}

// TeamRepo grants a team access to a repository at a specific permission level.
type TeamRepo struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	TeamID     uint   `json:"team_id" gorm:"not null;uniqueIndex:idx_team_repo"`
	RepoID     uint   `json:"repo_id" gorm:"not null;uniqueIndex:idx_team_repo"`
	Permission string `json:"permission" gorm:"not null;default:read"` // read, write, admin
}

func (TeamRepo) TableName() string { return "team_repos" }

type TeamRepoDTO struct {
	ID         uint   `json:"id"`
	TeamID     uint   `json:"team_id"`
	TeamName   string `json:"team_name"`
	RepoID     uint   `json:"repo_id"`
	Permission string `json:"permission"`
}

func (tr *TeamRepo) DTO() TeamRepoDTO {
	return TeamRepoDTO{
		ID:         tr.ID,
		TeamID:     tr.TeamID,
		RepoID:     tr.RepoID,
		Permission: tr.Permission,
	}
}
