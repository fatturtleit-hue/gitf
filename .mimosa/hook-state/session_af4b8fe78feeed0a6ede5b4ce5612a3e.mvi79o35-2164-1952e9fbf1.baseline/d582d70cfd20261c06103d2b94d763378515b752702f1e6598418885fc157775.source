package repo

import "gitf/internal/models"

// OwnerType constants for Repository.OwnerType.
const (
	OwnerTypeUser = "user"
	OwnerTypeOrg  = "org"
)

// RepositoryStore abstracts repository persistence and access control.
// Handlers depend on this interface, not on the concrete Service, enabling test mocking.
type RepositoryStore interface {
	Create(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool) (*models.Repository, error)
	Delete(ownerType string, ownerID uint, name string) error
	GetByOwnerAndName(ownerName, name string) (*models.Repository, error)
	Update(repo *models.Repository) error
	List(userID uint, search string, limit, offset int) ([]models.Repository, error)
	Count(userID uint, search string) (int, error)
	CanRead(repo *models.Repository, user *models.User) bool
	CanWrite(repo *models.Repository, user *models.User) bool
	CanAdmin(repo *models.Repository, user *models.User) bool

	// Fork creates a forked copy of an existing repository under a new owner.
	Fork(ownerType string, ownerID uint, name, description, defaultBranch string, isPrivate bool, forkedFromID uint) (*models.Repository, error)

	// ResolveOwner looks up an owner by username/orgname and returns its type, ID, and display name.
	ResolveOwner(ownerName string) (ownerType string, ownerID uint, ownerDisplayName string, err error)

	// Collaborator management
	ListCollaborators(repoID uint) ([]models.Collaborator, error)
	AddCollaborator(repoID uint, username string, permission string, quotaBytes int64) (*models.Collaborator, error)
	UpdateCollaborator(collabID uint, permission string, quotaBytes int64) (*models.Collaborator, error)
	RemoveCollaborator(repoID uint, userID uint) error
	CanReadWithCollab(repo *models.Repository, user *models.User) bool
	CountRepos() (int64, error)
	CountUsers() (int64, error)
}

// Compile-time check that Service satisfies RepositoryStore.
var _ RepositoryStore = (*Service)(nil)
