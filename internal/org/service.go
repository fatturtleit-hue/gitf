package org

import (
	"errors"
	"fmt"
	"strings"

	"gitf/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service implements Store with GORM.
type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

// ── Organization ──────────────────────────────────────

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// GORM v2 sentinel for duplicate keys.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// SQLite and some other drivers surface unique-constraint failures in the message.
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func (s *Service) Create(name, displayName, description string, ownerID uint) (*models.Organization, error) {
	org := models.Organization{
		Name:        name,
		DisplayName: displayName,
		Description: description,
	}
	// Use transaction: create org + add owner as member.
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: %v", ErrOrgConflict, err)
			}
			return fmt.Errorf("create organization: %w", err)
		}
		member := models.OrgMember{
			OrgID:  org.ID,
			UserID: ownerID,
			Role:   "owner",
		}
		if err := tx.Create(&member).Error; err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: %v", ErrOrgConflict, err)
			}
			return fmt.Errorf("add owner: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *Service) Update(orgID uint, displayName, description string) (*models.Organization, error) {
	var org models.Organization
	if err := s.DB.First(&org, orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("find organization: %w", err)
	}
	org.DisplayName = displayName
	org.Description = description
	if err := s.DB.Save(&org).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: %v", ErrOrgConflict, err)
		}
		return nil, fmt.Errorf("update organization: %w", err)
	}
	return &org, nil
}

func (s *Service) GetByName(name string) (*models.Organization, error) {
	var org models.Organization
	if err := s.DB.Where("name = ?", name).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (s *Service) Delete(orgID uint) error {
	return s.DB.Delete(&models.Organization{}, orgID).Error
}

func (s *Service) List(userID uint) ([]models.Organization, error) {
	// List orgs where user is a member.
	var orgIDs []uint
	s.DB.Model(&models.OrgMember{}).
		Where("user_id = ?", userID).
		Pluck("org_id", &orgIDs)
	if len(orgIDs) == 0 {
		return []models.Organization{}, nil
	}
	var orgs []models.Organization
	if err := s.DB.Where("id IN ?", orgIDs).Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

// ── Members ────────────────────────────────────────────

func (s *Service) AddMember(orgID, userID uint, role string) (*models.OrgMember, error) {
	m := models.OrgMember{
		OrgID:  orgID,
		UserID: userID,
		Role:   role,
	}
	if err := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error; err != nil {
		return nil, err
	}
	// Load the user association if the row was created.
	if m.ID == 0 {
		s.DB.Where("org_id = ? AND user_id = ?", orgID, userID).Preload("User").First(&m)
	}
	return &m, nil
}

func (s *Service) UpdateMember(memberID uint, role string) (*models.OrgMember, error) {
	var m models.OrgMember
	if err := s.DB.Preload("User").First(&m, memberID).Error; err != nil {
		return nil, err
	}
	m.Role = role
	if err := s.DB.Omit(clause.Associations).Save(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) RemoveMember(orgID, userID uint) error {
	return s.DB.Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&models.OrgMember{}).Error
}

func (s *Service) ListMembers(orgID uint) ([]models.OrgMember, error) {
	var members []models.OrgMember
	if err := s.DB.Preload("User").Where("org_id = ?", orgID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (s *Service) GetMember(orgID, userID uint) (*models.OrgMember, error) {
	var m models.OrgMember
	if err := s.DB.Preload("User").Where("org_id = ? AND user_id = ?", orgID, userID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotOrgMember
		}
		return nil, err
	}
	return &m, nil
}

// ── Teams ──────────────────────────────────────────────

func (s *Service) CreateTeam(orgID uint, name, description string) (*models.Team, error) {
	t := models.Team{
		OrgID:       orgID,
		Name:        name,
		Description: description,
	}
	if err := s.DB.Create(&t).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTeamConflict, err)
	}
	return &t, nil
}

func (s *Service) GetTeam(teamID uint) (*models.Team, error) {
	var t models.Team
	if err := s.DB.First(&t, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (s *Service) DeleteTeam(teamID uint) error {
	return s.DB.Delete(&models.Team{}, teamID).Error
}

func (s *Service) ListTeams(orgID uint) ([]models.Team, error) {
	var teams []models.Team
	if err := s.DB.Where("org_id = ?", orgID).Find(&teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

// ── Team members ───────────────────────────────────────

func (s *Service) AddTeamMember(teamID, userID uint) (*models.TeamMember, error) {
	m := models.TeamMember{
		TeamID: teamID,
		UserID: userID,
	}
	if err := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error; err != nil {
		return nil, err
	}
	if m.ID == 0 {
		s.DB.Where("team_id = ? AND user_id = ?", teamID, userID).Preload("User").First(&m)
	}
	return &m, nil
}

func (s *Service) RemoveTeamMember(teamID, userID uint) error {
	return s.DB.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&models.TeamMember{}).Error
}

func (s *Service) ListTeamMembers(teamID uint) ([]models.TeamMember, error) {
	var members []models.TeamMember
	if err := s.DB.Preload("User").Where("team_id = ?", teamID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// ── Team repos ─────────────────────────────────────────

func (s *Service) AddTeamRepo(teamID, repoID uint, permission string) (*models.TeamRepo, error) {
	tr := models.TeamRepo{
		TeamID:     teamID,
		RepoID:     repoID,
		Permission: permission,
	}
	if err := s.DB.Create(&tr).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTeamRepoConflict, err)
	}
	return &tr, nil
}

func (s *Service) RemoveTeamRepo(teamRepoID uint) error {
	return s.DB.Delete(&models.TeamRepo{}, teamRepoID).Error
}

func (s *Service) ListTeamRepos(teamID uint) ([]models.TeamRepo, error) {
	var repos []models.TeamRepo
	if err := s.DB.Where("team_id = ?", teamID).Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}
