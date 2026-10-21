package db

import (
	"testing"

	"gitf/internal/config"
)

func TestNewSQLite(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "file::memory:"},
	}
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestNewUnsupportedDriver(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "mysql", DSN: "root@tcp(localhost)/gitf"},
	}
	if _, err := New(cfg); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestMigrate(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", DSN: "file::memory:"},
	}
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}
