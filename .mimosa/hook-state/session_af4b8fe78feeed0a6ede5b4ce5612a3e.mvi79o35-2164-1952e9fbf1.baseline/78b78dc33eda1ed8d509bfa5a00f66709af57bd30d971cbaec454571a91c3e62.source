package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear relevant env vars for deterministic test.
	for _, k := range []string{
		"ADDR", "DATA_DIR", "DB_DRIVER", "DB_DSN", "JWT_SECRET",
		"LOG_LEVEL", "LOG_FORMAT", "TLS_ENABLED",
		"DISABLE_REGISTRATION", "DEFAULT_ADMIN_USERNAME", "DEFAULT_ADMIN_PASSWORD",
	} {
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("expected Addr=:8080, got %s", cfg.Addr)
	}
	if cfg.DataDir != "./data" {
		t.Errorf("expected DataDir=./data, got %s", cfg.DataDir)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("expected DB Driver=sqlite, got %s", cfg.Database.Driver)
	}
	if cfg.JWTSecret == "" {
		t.Error("expected JWTSecret to be auto-generated, got empty")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected LogLevel=info, got %s", cfg.Log.Level)
	}
	if cfg.DisableRegistration {
		t.Error("expected DisableRegistration=false by default")
	}
	if cfg.DefaultAdminUsername != "admin" {
		t.Errorf("expected DefaultAdminUsername=admin, got %s", cfg.DefaultAdminUsername)
	}
	if cfg.DefaultAdminPassword != "" {
		t.Errorf("expected DefaultAdminPassword empty by default, got %s", cfg.DefaultAdminPassword)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	os.Setenv("ADDR", ":9090")
	os.Setenv("JWT_SECRET", "my-fixed-secret")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DISABLE_REGISTRATION", "true")
	os.Setenv("DEFAULT_ADMIN_USERNAME", "root")
	os.Setenv("DEFAULT_ADMIN_PASSWORD", "change-me")
	defer func() {
		os.Unsetenv("ADDR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("DB_DRIVER")
		os.Unsetenv("DB_DSN")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("DISABLE_REGISTRATION")
		os.Unsetenv("DEFAULT_ADMIN_USERNAME")
		os.Unsetenv("DEFAULT_ADMIN_PASSWORD")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("expected Addr=:9090, got %s", cfg.Addr)
	}
	if cfg.JWTSecret != "my-fixed-secret" {
		t.Errorf("expected JWTSecret=my-fixed-secret, got %s", cfg.JWTSecret)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected LogLevel=debug, got %s", cfg.Log.Level)
	}
	if !cfg.DisableRegistration {
		t.Error("expected DisableRegistration=true")
	}
	if cfg.DefaultAdminUsername != "root" {
		t.Errorf("expected DefaultAdminUsername=root, got %s", cfg.DefaultAdminUsername)
	}
	if cfg.DefaultAdminPassword != "change-me" {
		t.Errorf("expected DefaultAdminPassword=change-me, got %s", cfg.DefaultAdminPassword)
	}
}

func TestReposDir(t *testing.T) {
	cfg := &Config{DataDir: "/data"}
	cfg.reposDir = cfg.DataDir + "/repositories"
	if cfg.ReposDir() != "/data/repositories" {
		t.Errorf("expected /data/repositories, got %s", cfg.ReposDir())
	}
}

func TestJWTSecretAutoGenerate(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	defer func() {
		os.Unsetenv("ADDR")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("DB_DRIVER")
		os.Unsetenv("DB_DSN")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("DISABLE_REGISTRATION")
		os.Unsetenv("DEFAULT_ADMIN_USERNAME")
		os.Unsetenv("DEFAULT_ADMIN_PASSWORD")
	}()

	cfg, _ := Load()
	if len(cfg.JWTSecret) != 64 {
		t.Errorf("expected 64-char hex secret, got %d chars", len(cfg.JWTSecret))
	}

	cfg2, _ := Load()
	if cfg.JWTSecret == cfg2.JWTSecret {
		t.Error("expected different random secrets across Load() calls")
	}
}
