package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gitf/internal/config"
	"gitf/internal/gitutil"
	"gitf/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Sentinel errors returned by Service. Handlers compare with errors.Is to
// translate to HTTP status codes — never on string matching.
var (
	// ErrNotFound is returned when a repository does not exist.
	ErrNotFound = errors.New("repository not found")
	// ErrInvalidName is returned when the repository name fails validation.
	ErrInvalidName = errors.New("invalid repository name")
	// ErrOwnerNotFound is returned when the configured owner does not exist.
	ErrOwnerNotFound = errors.New("owner not found")
	// ErrConflict is returned when a repository with the same owner+name exists.
	ErrConflict = errors.New("repository already exists")
)

// Service provides repository business logic.
type Service struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewService creates a new repository service.
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{DB: db, Config: cfg}
}

// ── Owner resolution ──────────────────────────────────

// ResolveOwner looks up an owner name in users then organizations, returning
// the owner type, database ID, and display name (username or org display name).
func (s *Service) ResolveOwner(ownerName string) (ownerType string, ownerID uint, ownerDisplayName string, err error) {
	var user models.User
	if err := s.DB.Where("username = ?", ownerName).First(&user).Error; err == nil {
		return OwnerTypeUser, user.ID, user.Username, nil
	}
	var org models.Organization
	if err := s.DB.Where("name = ?", ownerName).First(&org).Error; err == nil {
		return OwnerTypeOrg, org.ID, org.Name, nil
	}
	return "", 0, "", ErrOwnerNotFound
}

// ── Create / Delete / Query ───────────────────────────

// Create initializes a new repository for the given owner.
func (s *Service) Create(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool) (*models.Repository, error) {
	if !gitutil.ValidateName(name) {
		return nil, ErrInvalidName
	}

	// Validate owner exists and get display name for filesystem path.
	displayName, err := s.getDisplayName(ownerType, ownerID)
	if err != nil {
		return nil, ErrOwnerNotFound
	}

	repo := models.Repository{
		OwnerType:     ownerType,
		OwnerID:       ownerID,
		Name:          name,
		Description:   description,
		IsPrivate:     isPrivate,
		DefaultBranch: defaultBranch,
	}
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	if err := s.DB.Create(&repo).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}

	if err := gitutil.InitBare(s.Config, displayName, name, repo.DefaultBranch); err != nil {
		if rbErr := s.DB.Delete(&repo).Error; rbErr != nil {
			return nil, fmt.Errorf("init bare repo: %w; rollback also failed: %v (orphan row id=%d)", err, rbErr, repo.ID)
		}
		return nil, fmt.Errorf("init bare repo: %w", err)
	}

	// Populate Owner.Username for DTO compatibility.
	repo.Owner.Username = displayName
	return &repo, nil
}

// getDisplayName returns the human-readable name for an owner (username or org name).
func (s *Service) getDisplayName(ownerType string, ownerID uint) (string, error) {
	if ownerType == OwnerTypeUser {
		var user models.User
		if err := s.DB.First(&user, ownerID).Error; err != nil {
			return "", err
		}
		return user.Username, nil
	}
	var org models.Organization
	if err := s.DB.First(&org, ownerID).Error; err != nil {
		return "", err
	}
	return org.Name, nil
}

// Delete removes a repository by owner type, ID, and name.
func (s *Service) Delete(ownerType string, ownerID uint, name string) error {
	var repo models.Repository
	err := s.DB.Where("owner_type = ? AND owner_id = ? AND name = ?", ownerType, ownerID, name).First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("load repo: %w", err)
	}

	displayName, err := s.getDisplayName(ownerType, ownerID)
	if err != nil {
		return fmt.Errorf("load owner: %w", err)
	}

	if err := s.DB.Delete(&repo).Error; err != nil {
		return fmt.Errorf("delete repo row: %w", err)
	}

	return gitutil.RemoveBare(s.Config, displayName, name)
}

// GetByOwnerAndName loads a repository by owner name and repository name.
// Works for both user-owned and org-owned repos.
func (s *Service) GetByOwnerAndName(ownerName, name string) (*models.Repository, error) {
	ownerType, ownerID, displayName, err := s.ResolveOwner(ownerName)
	if err != nil {
		return nil, ErrNotFound
	}

	var repo models.Repository
	err = s.DB.Where("owner_type = ? AND owner_id = ? AND name = ?", ownerType, ownerID, name).
		Preload("ForkedFrom").
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("load repo by owner+name: %w", err)
	}

	// Populate Owner.Username for DTO and gitutil path compatibility.
	repo.Owner.Username = displayName
	return &repo, nil
}

// ── Visibility & Listing ─────────────────────────────

func (s *Service) visibilityFilter(query *gorm.DB, userID uint, search string) *gorm.DB {
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("repositories.name LIKE ? OR repositories.description LIKE ?", like, like)
	}
	if userID == 0 {
		query = query.Where("repositories.is_private = ?", false)
	} else {
		// Show public repos, user-owned repos, and org repos where the user is a member.
		query = query.Where(`repositories.is_private = ? OR
			(repositories.owner_type = ? AND repositories.owner_id = ?) OR
			(repositories.owner_type = ? AND repositories.owner_id IN (
				SELECT org_id FROM org_members WHERE user_id = ?
			))`,
			false,
			OwnerTypeUser, userID,
			OwnerTypeOrg, userID,
		)
	}
	return query
}

// List returns repositories visible to a user.
func (s *Service) List(userID uint, search string, limit, offset int) ([]models.Repository, error) {
	query := s.visibilityFilter(s.DB.Model(&models.Repository{}), userID, search)
	var repos []models.Repository
	if err := query.Order("repositories.updated_at DESC").
		Preload("ForkedFrom").
		Limit(limit).Offset(offset).Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	// Populate Owner.Username for each repo's DTO.
	for i := range repos {
		if repos[i].OwnerType == OwnerTypeUser {
			var user models.User
			if err := s.DB.First(&user, repos[i].OwnerID).Error; err == nil {
				repos[i].Owner.Username = user.Username
			}
		} else {
			var org models.Organization
			if err := s.DB.First(&org, repos[i].OwnerID).Error; err == nil {
				repos[i].Owner.Username = org.Name
			}
		}
	}
	return repos, nil
}

// Count returns the total number of repositories matching the same visibility rules as List.
func (s *Service) Count(userID uint, search string) (int, error) {
	query := s.visibilityFilter(s.DB.Model(&models.Repository{}), userID, search)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count repos: %w", err)
	}
	return int(count), nil
}

// ── Access control ────────────────────────────────────

// effectivePermission returns the highest permission the user has on the repo:
// "" (none), "read", "write", or "admin".
func (s *Service) effectivePermission(repo *models.Repository, user *models.User) string {
	// 1. Site admin → full access.
	if user != nil && user.IsSiteAdmin {
		return "admin"
	}
	if user == nil {
		if !repo.IsPrivate {
			return "read"
		}
		return ""
	}

	// 2. User-owner: check direct ownership.
	if repo.OwnerType == OwnerTypeUser && user.ID == repo.OwnerID {
		return "admin"
	}
	// 3. Org-owner: check if user is org owner.
	if repo.OwnerType == OwnerTypeOrg {
		if s.isOrgOwner(repo.OwnerID, user.ID) {
			return "admin"
		}
		// 4. Team-based access.
		if perm := s.getTeamPermission(repo.ID, user.ID); perm != "" {
			return perm
		}
	}
	// 5. Direct collaborator access.
	if perm := s.getCollaboratorPermission(repo.ID, user.ID); perm != "" {
		return perm
	}
	// 6. Public read floor.
	if !repo.IsPrivate {
		return "read"
	}
	return ""
}

// isOrgOwner checks whether a user is the owner of an organization.
func (s *Service) isOrgOwner(orgID, userID uint) bool {
	var count int64
	s.DB.Model(&models.OrgMember{}).
		Where("org_id = ? AND user_id = ? AND role = ?", orgID, userID, "owner").
		Count(&count)
	return count > 0
}

// getTeamPermission returns the highest permission granted via team membership.
func (s *Service) getTeamPermission(repoID, userID uint) string {
	var maxPerm string
	s.DB.Raw(`
		SELECT MAX(tr.permission) FROM team_repos tr
		JOIN team_members tm ON tm.team_id = tr.team_id
		WHERE tr.repo_id = ? AND tm.user_id = ?
	`, repoID, userID).Scan(&maxPerm)
	return maxPerm
}

// getCollaboratorPermission returns the user's direct collaborator permission.
func (s *Service) getCollaboratorPermission(repoID, userID uint) string {
	var perm string
	s.DB.Model(&models.Collaborator{}).
		Where("repo_id = ? AND user_id = ?", repoID, userID).
		Select("permission").
		Scan(&perm)
	return perm
}

// CanRead reports whether the user can read the repository.
func (s *Service) CanRead(repo *models.Repository, user *models.User) bool {
	perm := s.effectivePermission(repo, user)
	return perm == "read" || perm == "write" || perm == "admin"
}

// CanWrite reports whether the user can write (push) to the repository.
func (s *Service) CanWrite(repo *models.Repository, user *models.User) bool {
	perm := s.effectivePermission(repo, user)
	return perm == "write" || perm == "admin"
}

// CanAdmin reports whether the user can administer the repository.
func (s *Service) CanAdmin(repo *models.Repository, user *models.User) bool {
	perm := s.effectivePermission(repo, user)
	return perm == "admin"
}

// ── Fork ──────────────────────────────────────────────

// Fork creates a fork of an existing repository.
func (s *Service) Fork(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool, forkedFromID uint) (*models.Repository, error) {
	if !gitutil.ValidateName(name) {
		return nil, ErrInvalidName
	}

	displayName, err := s.getDisplayName(ownerType, ownerID)
	if err != nil {
		return nil, ErrOwnerNotFound
	}

	repo := models.Repository{
		OwnerType:     ownerType,
		OwnerID:       ownerID,
		Name:          name,
		Description:   description,
		IsPrivate:     isPrivate,
		DefaultBranch: defaultBranch,
		ForkedFromID:  &forkedFromID,
	}
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	if err := s.DB.Create(&repo).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}

	// Clone the source repo to create the fork.
	var source models.Repository
	if err := s.DB.First(&source, forkedFromID).Error; err != nil {
		s.DB.Delete(&repo)
		return nil, fmt.Errorf("source repo not found: %w", err)
	}
	// Owner is not preloaded on source — resolve the on-disk owner segment
	// from the store instead of the empty association.
	sourceOwner, err := s.getDisplayName(source.OwnerType, source.OwnerID)
	if err != nil {
		s.DB.Delete(&repo)
		return nil, fmt.Errorf("source owner not found: %w", err)
	}
	sourcePath := gitutil.RepoPath(s.Config, sourceOwner, source.Name)
	destPath := gitutil.RepoPath(s.Config, displayName, name)
	if err := cloneBare(sourcePath, destPath); err != nil {
		s.DB.Delete(&repo)
		return nil, fmt.Errorf("clone fork: %w", err)
	}

	// Invalidate cache for the new repo to force fresh open.
	gitutil.InvalidateRepoCache(displayName, name)

	repo.Owner.Username = displayName
	return &repo, nil
}

// cloneBare clones a bare repository using git clone --bare then copies HEAD.
func cloneBare(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create fork parent dir: %w", err)
	}
	cmd := exec.Command("git", "clone", "--bare", srcPath, dstPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone --bare: %w: %s", err, string(out))
	}
	return nil
}

// ── Update ────────────────────────────────────────────

// Update saves changes to an existing repository.
func (s *Service) Update(repo *models.Repository) error {
	// Omit associations: Owner/ForkedFrom may be partially populated (e.g.
	// Owner.Username filled in without a primary key) and a plain Save would
	// rewrite owner_id to the association's zero key, detaching the repo.
	return s.DB.Omit(clause.Associations).Save(repo).Error
}

// ── Stats ─────────────────────────────────────────────

// CountRepos returns the total number of repositories in the system.
func (s *Service) CountRepos() (int64, error) {
	var count int64
	err := s.DB.Model(&models.Repository{}).Count(&count).Error
	return count, err
}

// CountUsers returns the total number of registered users.
func (s *Service) CountUsers() (int64, error) {
	var count int64
	err := s.DB.Model(&models.User{}).Count(&count).Error
	return count, err
}

// ── Collaborators ────────────────────────────────────

func (s *Service) hasCollaboratorRole(repoID, userID uint, roles ...string) bool {
	var count int64
	s.DB.Model(&models.Collaborator{}).
		Where("repo_id = ? AND user_id = ? AND permission IN ?", repoID, userID, roles).
		Count(&count)
	return count > 0
}

func (s *Service) ListCollaborators(repoID uint) ([]models.Collaborator, error) {
	var collabs []models.Collaborator
	if err := s.DB.Preload("User").Where("repo_id = ?", repoID).Find(&collabs).Error; err != nil {
		return nil, err
	}
	return collabs, nil
}

func (s *Service) AddCollaborator(repoID uint, username string, permission string, quotaBytes int64) (*models.Collaborator, error) {
	var user models.User
	if err := s.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	c := models.Collaborator{
		RepoID:     repoID,
		UserID:     user.ID,
		Permission: permission,
		QuotaBytes: quotaBytes,
	}
	if err := s.DB.Create(&c).Error; err != nil {
		return nil, err
	}
	c.User = user
	return &c, nil
}

func (s *Service) UpdateCollaborator(collabID uint, permission string, quotaBytes int64) (*models.Collaborator, error) {
	var c models.Collaborator
	if err := s.DB.Preload("User").First(&c, collabID).Error; err != nil {
		return nil, err
	}
	c.Permission = permission
	c.QuotaBytes = quotaBytes
	if err := s.DB.Omit(clause.Associations).Save(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) RemoveCollaborator(repoID uint, userID uint) error {
	return s.DB.Where("repo_id = ? AND user_id = ?", repoID, userID).Delete(&models.Collaborator{}).Error
}

func (s *Service) CanReadWithCollab(repo *models.Repository, user *models.User) bool {
	return s.CanRead(repo, user)
}
