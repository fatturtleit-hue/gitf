package handlers

import (
	"net/http"
	"os"
	"testing"

	"gitf/internal/auth"
	"gitf/internal/config"
	"gitf/internal/issues"
	"gitf/internal/middleware"
	"gitf/internal/models"
	"gitf/internal/org"
	"gitf/internal/repo"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// htEnv is a full-stack handler test environment: in-memory SQLite with the
// real GORM stores (not mocks) so cross-layer regressions surface here.
type htEnv struct {
	db     *gorm.DB
	cfg    *config.Config
	users  *gormUserStore
	repos  repo.RepositoryStore
	orgs   *org.Service
	issues issues.Store
}

func migrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Collaborator{},
		&models.AuditLog{},
		&models.Organization{},
		&models.OrgMember{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamRepo{},
		&models.Release{},
		&models.ReleaseAsset{},
		&models.BranchProtection{},
		&models.Issue{},
		&models.Comment{},
		&models.Label{},
		&models.IssueLabel{},
		&models.Milestone{},
		&models.IssueColumn{},
		&models.IssueColumnItem{},
		&models.PullRequest{},
		&models.Review{},
		&models.PersonalAccessToken{},
		&models.Webhook{},
		&models.WebhookDelivery{},
	)
}

func newHandlerEnv(t *testing.T) *htEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := migrateAll(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dataDir := t.TempDir()
	os.Setenv("DATA_DIR", dataDir)
	os.Setenv("JWT_SECRET", "test-secret-for-handlers")
	os.Unsetenv("DB_DSN")
	os.Unsetenv("HOOK_SECRET")
	t.Cleanup(func() {
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("HOOK_SECRET")
	})
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	env := &htEnv{
		db:     db,
		cfg:    cfg,
		users:  NewGormUserStore(db).(*gormUserStore),
		repos:  repo.NewService(db, cfg),
		orgs:   org.NewService(db),
		issues: issues.NewService(db),
	}

	// alice owns everything, bob is the other guy.
	for _, name := range []string{"alice", "bob"} {
		hash, err := auth.HashPassword(testPassword())
		if err != nil {
			t.Fatal(err)
		}
		u := &models.User{Username: name, Email: name + "@test.local", PasswordHash: hash}
		if err := env.users.Create(t.Context(), u); err != nil {
			t.Fatalf("seed user %s: %v", name, err)
		}
	}
	return env
}

// user fetches a seeded user by username.
func (e *htEnv) user(t *testing.T, username string) *models.User {
	t.Helper()
	u, err := e.users.ByUsername(t.Context(), username)
	if err != nil {
		t.Fatalf("seeded user %s missing: %v", username, err)
	}
	return u
}

// asUser injects the authenticated-user context into a request.
func asUser(r *http.Request, u *models.User) *http.Request {
	return r.WithContext(middleware.WithUser(r.Context(), u))
}

// seedRepo creates a real repository row (and its on-disk bare repo) owned by
// alice, so handlers that touch gitutil paths work.
func (e *htEnv) seedRepo(t *testing.T, name string, isPrivate bool) *models.Repository {
	t.Helper()
	r, err := e.repos.Create(repo.OwnerTypeUser, e.user(t, "alice").ID, name, "", "main", isPrivate)
	if err != nil {
		t.Fatalf("seed repo %s: %v", name, err)
	}
	return r
}
