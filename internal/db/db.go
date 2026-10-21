package db

import (
	"fmt"
	"time"

	"gitf/internal/config"
	"gitf/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// New initializes a GORM database connection based on the configuration.
func New(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Database.Driver {
	case "sqlite":
		// glebarez/sqlite wraps modernc.org/sqlite so the build stays
		// CGO-free. Using this driver instead of gorm.io/driver/sqlite
		// avoids pulling the CGO-only mattn/go-sqlite3 dependency.
		dialector = sqlite.Open(cfg.Database.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

// Migrate runs GORM auto-migrations for all models.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
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
		&models.Workflow{},
		&models.WorkflowRun{},
		&models.WorkflowRunLog{},
		&models.Runner{},
		&models.Package{},
		&models.PackageVersion{},
		&models.PackageFile{},
	)
	if err != nil {
		return err
	}
	return migrateIndexes(db)
}
// migrateIndexes repairs indexes that earlier model definitions created
// wrongly. The releases uniqueness used to cover only tag_name, which wrongly
// forbade the same tag name in two repositories.
func migrateIndexes(db *gorm.DB) error {
	// Drop-and-recreate is idempotent: fresh databases get the correct index
	// from AutoMigrate and this is a no-op rebuild.
	if err := db.Exec("DROP INDEX IF EXISTS idx_repo_tag").Error; err != nil {
		return fmt.Errorf("drop release tag index: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_repo_tag ON releases (repo_id, tag_name)").Error; err != nil {
		return fmt.Errorf("recreate release tag index: %w", err)
	}
	return nil
}
