package repo

import (
	"errors"
	"testing"

	"gitf/internal/config"
	"gitf/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupRepoStore creates an in-memory GORM-backed repo store for tests.
// We intentionally use glebarez/sqlite so tests do NOT require CGO
// (matching the production build which sets CGO_ENABLED=0).
func setupRepoStore(t *testing.T) (*Service, *config.Config) {
	t.Helper()
	// We use glebarez/sqlite for tests so they stay CGO-free and match the
	// production build (CGO_ENABLED=0).
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Organization{},
		&models.OrgMember{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamRepo{},
		&models.Collaborator{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cfg := &config.Config{}
	return NewService(db, cfg), cfg
}

func TestCreateAndGet(t *testing.T) {
	svc, _ := setupRepoStore(t)
	user := createUser(t, svc, "alice")

	repo, err := svc.Create(OwnerTypeUser, user.ID, "hello", "desc", "main", false)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if repo.Name != "hello" {
		t.Errorf("name: got %q, want hello", repo.Name)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("default branch: got %q, want main", repo.DefaultBranch)
	}

	got, err := svc.GetByOwnerAndName("alice", "hello")
	if err != nil {
		t.Fatalf("get repo: %v", err)
	}
	if got.Owner.Username != "alice" {
		t.Errorf("owner username: got %q, want alice", got.Owner.Username)
	}
}

func TestCreateInvalidName(t *testing.T) {
	svc, _ := setupRepoStore(t)
	user := createUser(t, svc, "bob")
	_, err := svc.Create(OwnerTypeUser, user.ID, "../evil", "", "main", false)
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestCreateOwnerNotFound(t *testing.T) {
	svc, _ := setupRepoStore(t)
	_, err := svc.Create(OwnerTypeUser, 999, "hello", "", "main", false)
	if !errors.Is(err, ErrOwnerNotFound) {
		t.Fatalf("expected ErrOwnerNotFound, got %v", err)
	}
}

func TestVisibility(t *testing.T) {
	svc, _ := setupRepoStore(t)
	alice := createUser(t, svc, "alice")
	bob := createUser(t, svc, "bob")
	publicRepo, _ := svc.Create(OwnerTypeUser, alice.ID, "public", "", "main", false)
	privateRepo, _ := svc.Create(OwnerTypeUser, alice.ID, "private", "", "main", true)

	tests := []struct {
		name   string
		repo   *models.Repository
		viewer *models.User
		want   bool
	}{
		{"public anonymous", publicRepo, nil, true},
		{"public other user", publicRepo, bob, true},
		{"private anonymous", privateRepo, nil, false},
		{"private owner", privateRepo, alice, true},
		{"private other user", privateRepo, bob, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.CanRead(tt.repo, tt.viewer); got != tt.want {
				t.Errorf("CanRead = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteAccess(t *testing.T) {
	svc, _ := setupRepoStore(t)
	alice := createUser(t, svc, "alice")
	bob := createUser(t, svc, "bob")
	repo, _ := svc.Create(OwnerTypeUser, alice.ID, "mine", "", "main", false)

	if !svc.CanWrite(repo, alice) {
		t.Error("owner should be able to write")
	}
	if svc.CanWrite(repo, bob) {
		t.Error("other user should not be able to write")
	}
	if svc.CanWrite(repo, nil) {
		t.Error("anonymous should not be able to write")
	}
}

func TestListVisibility(t *testing.T) {
	svc, _ := setupRepoStore(t)
	alice := createUser(t, svc, "alice")
	bob := createUser(t, svc, "bob")
	svc.Create(OwnerTypeUser, alice.ID, "alpha", "", "main", false)
	svc.Create(OwnerTypeUser, alice.ID, "beta", "", "main", true)
	svc.Create(OwnerTypeUser, bob.ID, "gamma", "", "main", false)

	// Anonymous sees only public repos (2).
	anon, _, err := listAndCount(svc, 0, "", 10, 0)
	if err != nil {
		t.Fatalf("list anon: %v", err)
	}
	if len(anon) != 2 {
		t.Errorf("anon saw %d repos, want 2", len(anon))
	}

	// Alice sees her own + all public repos (3).
	aliceRepos, count, err := listAndCount(svc, alice.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("list alice: %v", err)
	}
	if count != 3 {
		t.Errorf("alice count = %d, want 3", count)
	}
	if len(aliceRepos) != 3 {
		t.Errorf("alice list len = %d, want 3", len(aliceRepos))
	}

	// Search by name works and respects visibility.
	found, count, err := listAndCount(svc, bob.ID, "beta", 10, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if count != 0 || len(found) != 0 {
		t.Errorf("bob should not see alice's private 'beta' repo: count=%d len=%d", count, len(found))
	}
}

func TestDelete(t *testing.T) {
	svc, _ := setupRepoStore(t)
	alice := createUser(t, svc, "alice")
	_, _ = svc.Create(OwnerTypeUser, alice.ID, "bye", "", "main", false)

	if err := svc.Delete(OwnerTypeUser, alice.ID, "bye"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := svc.GetByOwnerAndName("alice", "bye")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc, _ := setupRepoStore(t)
	user := createUser(t, svc, "carol")
	err := svc.Delete(OwnerTypeUser, user.ID, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func createUser(t *testing.T, svc *Service, username string) *models.User {
	t.Helper()
	u := models.User{Username: username, Email: username + "@example.com", PasswordHash: "x"}
	if err := svc.DB.Create(&u).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return &u
}

func listAndCount(svc *Service, userID uint, search string, limit, offset int) ([]models.Repository, int, error) {
	repos, err := svc.List(userID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	count, err := svc.Count(userID, search)
	if err != nil {
		return nil, 0, err
	}
	return repos, count, nil
}
