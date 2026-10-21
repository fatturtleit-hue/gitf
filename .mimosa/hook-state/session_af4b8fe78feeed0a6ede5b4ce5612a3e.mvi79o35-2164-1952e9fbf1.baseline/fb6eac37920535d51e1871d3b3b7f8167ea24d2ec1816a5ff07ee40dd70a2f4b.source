package org

import (
	"errors"

	"gitf/internal/models"
)

// Sentinel errors.
var (
	ErrOrgNotFound     = errors.New("organization not found")
	ErrOrgConflict     = errors.New("organization already exists")
	ErrNotOrgMember    = errors.New("user is not a member of the organization")
	ErrTeamNotFound    = errors.New("team not found")
	ErrTeamConflict    = errors.New("team already exists")
	ErrTeamRepoConflict = errors.New("team already has access to this repository")
)

// Store abstracts organization persistence.
type Store interface {
	Create(name, displayName, description string, ownerID uint) (*models.Organization, error)
	Update(orgID uint, displayName, description string) (*models.Organization, error)
	GetByName(name string) (*models.Organization, error)
	Delete(orgID uint) error
	List(userID uint) ([]models.Organization, error)

	// Members
	AddMember(orgID, userID uint, role string) (*models.OrgMember, error)
	UpdateMember(memberID uint, role string) (*models.OrgMember, error)
	RemoveMember(orgID, userID uint) error
	ListMembers(orgID uint) ([]models.OrgMember, error)
	GetMember(orgID, userID uint) (*models.OrgMember, error)

	// Teams
	CreateTeam(orgID uint, name, description string) (*models.Team, error)
	GetTeam(teamID uint) (*models.Team, error)
	DeleteTeam(teamID uint) error
	ListTeams(orgID uint) ([]models.Team, error)

	// Team members
	AddTeamMember(teamID, userID uint) (*models.TeamMember, error)
	RemoveTeamMember(teamID, userID uint) error
	ListTeamMembers(teamID uint) ([]models.TeamMember, error)

	// Team repos
	AddTeamRepo(teamID, repoID uint, permission string) (*models.TeamRepo, error)
	RemoveTeamRepo(teamRepoID uint) error
	ListTeamRepos(teamID uint) ([]models.TeamRepo, error)
}

// Compile-time check.
var _ Store = (*Service)(nil)
