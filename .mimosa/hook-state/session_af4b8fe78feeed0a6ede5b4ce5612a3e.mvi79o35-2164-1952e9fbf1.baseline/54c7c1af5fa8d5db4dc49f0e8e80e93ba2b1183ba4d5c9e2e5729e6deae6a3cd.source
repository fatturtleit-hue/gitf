package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Addr      string         `env:"ADDR" envDefault:":8080"`
	DataDir   string         `env:"DATA_DIR" envDefault:"./data"`
	Database  DatabaseConfig `envPrefix:"DB_"`
	JWTSecret string         `env:"JWT_SECRET"`
	Log       LogConfig      `envPrefix:"LOG_"`
	TLS       TLSConfig      `envPrefix:"TLS_"`
	// ArchiveMaxBytes limits the size of git archive requests (0 = no limit).
	ArchiveMaxBytes int64 `env:"ARCHIVE_MAX_BYTES" envDefault:"0"`
	// OrgAllowCreate controls whether users can create organizations.
	OrgAllowCreate bool `env:"ORG_ALLOW_CREATE" envDefault:"true"`
	// DisableRegistration disables public self-registration. When true, only
	// site administrators can create new accounts.
	DisableRegistration bool `env:"DISABLE_REGISTRATION" envDefault:"false"`
	// DefaultAdminUsername is the bootstrap admin account created on first run.
	DefaultAdminUsername string `env:"DEFAULT_ADMIN_USERNAME" envDefault:"admin"`
	// DefaultAdminPassword is the initial password for the bootstrap admin.
	// It must be set explicitly when DisableRegistration is true and the
	// database has no users; otherwise the server refuses to start.
	DefaultAdminPassword string `env:"DEFAULT_ADMIN_PASSWORD" envDefault:""`
	// WebhookWorkerCount controls the number of webhook delivery workers.
	WebhookWorkerCount int `env:"WEBHOOK_WORKERS" envDefault:"4"`
	// WebhookTimeoutSec is the HTTP request timeout for webhook deliveries.
	WebhookTimeoutSec int `env:"WEBHOOK_TIMEOUT" envDefault:"10"`
	// WebhookMaxRetries is the maximum number of delivery retries.
	WebhookMaxRetries int `env:"WEBHOOK_MAX_RETRIES" envDefault:"3"`
	// ActionsRunnerPollTimeoutSec is the long-poll timeout for runner job polling.
	ActionsRunnerPollTimeoutSec int `env:"ACTIONS_RUNNER_POLL_TIMEOUT" envDefault:"30"`
	// PackageMaxBytesMB is the maximum package upload size in MB (default 2048 = 2GB for Docker layers).
	PackageMaxBytesMB int64 `env:"PACKAGE_MAX_BYTES_MB" envDefault:"2048"`
	// HookSecret is a shared secret used by pre-receive hooks to authenticate
	// internal API calls for branch-protection enforcement.
	HookSecret string `env:"HOOK_SECRET" envDefault:""`

	// Pre-computed paths (set by Load).
	reposDir string
}

// DatabaseConfig describes the database connection.
type DatabaseConfig struct {
	Driver string `env:"DRIVER" envDefault:"sqlite"`
	DSN    string `env:"DSN" envDefault:"./data/app.db"`
}

// LogConfig controls structured logging behavior.
type LogConfig struct {
	Level  string `env:"LEVEL" envDefault:"info"`
	Format string `env:"FORMAT" envDefault:"json"`
}

// TLSConfig holds optional TLS settings.
type TLSConfig struct {
	Enabled  bool   `env:"ENABLED" envDefault:"false"`
	CertFile string `env:"CERT_FILE"`
	KeyFile  string `env:"KEY_FILE"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if cfg.JWTSecret == "" {
		// Auto-generating a secret is convenient for local development but
		// catastrophic in production: every restart invalidates all tokens.
		// Print a visible warning to stderr before log is initialized.
		fmt.Fprintln(os.Stderr, "WARNING: JWT_SECRET is not set; a random secret has been generated. "+
			"Set JWT_SECRET to a fixed value or all users will be logged out on restart.")
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, fmt.Errorf("generate jwt secret: %w", err)
		}
		cfg.JWTSecret = hex.EncodeToString(secret)
	}
	cfg.reposDir = cfg.DataDir + "/repositories"
	return cfg, nil
}

// ReposDir returns the root directory where bare Git repositories are stored.
func (c *Config) ReposDir() string { return c.reposDir }
