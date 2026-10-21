package auth

import (
	"errors"
	"strings"
	"testing"

	"gitf/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.PersonalAccessToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestTokenStoreLifecycle(t *testing.T) {
	store := NewTokenStore(newTokenTestDB(t))

	plain, token, err := store.Create(7, "ci", "repo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(plain, "gitf_") {
		t.Errorf("token prefix = %q, want gitf_", plain[:5])
	}
	if len(plain) != len("gitf_")+64 {
		t.Errorf("token length = %d, want gitf_ + 64 hex chars", len(plain))
	}
	if token.UserID != 7 || token.Name != "ci" || token.TokenHash == plain {
		t.Errorf("stored token must be hashed, got %+v", token)
	}
	if token.Prefix != plain[:13] {
		t.Errorf("prefix = %q, want %q", token.Prefix, plain[:13])
	}

	// Lookup by the plain token resolves to the right record.
	found, err := store.Lookup(plain)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if found.ID != token.ID || found.UserID != 7 {
		t.Errorf("Lookup got %+v", found)
	}

	list, err := store.List(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != token.ID {
		t.Errorf("List = %+v", list)
	}

	// Other users see nothing.
	otherList, _ := store.List(8)
	if len(otherList) != 0 {
		t.Errorf("other user's list = %d, want 0", len(otherList))
	}

	// Deleting another user's token id is a no-op error.
	if err := store.Delete(8, token.ID); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("cross-user delete err = %v, want ErrTokenNotFound", err)
	}

	if err := store.Delete(7, token.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Lookup(plain); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("lookup after delete err = %v, want ErrTokenNotFound", err)
	}
	if err := store.Delete(7, token.ID); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("double delete err = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenStoreLookupInvalid(t *testing.T) {
	store := NewTokenStore(newTokenTestDB(t))
	if _, err := store.Lookup("gitf_deadbeef"); !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("unknown token err = %v, want ErrTokenNotFound", err)
	}
}
