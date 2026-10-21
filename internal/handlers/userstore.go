package handlers

import (
	"context"
	"errors"
	"fmt"

	"gitf/internal/gitutil"
	"gitf/internal/models"

	"gorm.io/gorm"
)

// UserStore abstracts persistence for users. Auth handlers and the auth
// middleware depend on this interface — never on *gorm.DB directly — so the
// auth path is unit-testable with simple in-memory mocks.
type UserStore interface {
	Create(ctx context.Context, u *models.User) error
	ByUsername(ctx context.Context, username string) (*models.User, error)
	ByID(ctx context.Context, id uint) (*models.User, error)
	List(ctx context.Context, search string, limit, offset int) ([]models.User, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, u *models.User) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context) (bool, error)
}

// Sentinel errors returned by UserStore implementations.
var (
	// ErrUserNotFound is returned when no row matches the lookup.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserConflict is returned when Create violates the unique index on
	// username or email. Translates to HTTP 409 at the handler layer.
	ErrUserConflict = errors.New("user already exists")
	// ErrLastAdmin is returned when an operation would remove the last site admin.
	ErrLastAdmin = errors.New("cannot remove the last site administrator")
)

// gormUserStore is the production UserStore backed by GORM.
type gormUserStore struct{ db *gorm.DB }

// NewGormUserStore returns a UserStore backed by the given GORM database.
func NewGormUserStore(db *gorm.DB) UserStore { return &gormUserStore{db: db} }

func (s *gormUserStore) Create(ctx context.Context, u *models.User) error {
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		// GORM doesn't expose a portable "duplicate key" sentinel, but on
		// this table the only Create error in normal operation is a unique
		// index violation on username or email.
		return fmt.Errorf("%w: %v", ErrUserConflict, err)
	}
	return nil
}

func (s *gormUserStore) ByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("lookup user by username: %w", err)
	}
	return &u, nil
}

func (s *gormUserStore) ByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	if err := s.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("lookup user by id: %w", err)
	}
	return &u, nil
}

func (s *gormUserStore) List(ctx context.Context, search string, limit, offset int) ([]models.User, error) {
	var users []models.User
	q := s.db.WithContext(ctx).Order("created_at desc")
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("username LIKE ? OR email LIKE ?", like, like)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *gormUserStore) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (s *gormUserStore) Update(ctx context.Context, u *models.User) error {
	if err := s.db.WithContext(ctx).Save(u).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *gormUserStore) Delete(ctx context.Context, id uint) error {
	if err := s.db.WithContext(ctx).Delete(&models.User{}, id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *gormUserStore) Exists(ctx context.Context) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check users exist: %w", err)
	}
	return count > 0, nil
}

// validUsername reuses the repo-name regex so a user's identifier survives
// being embedded in URL paths without re-encoding.
func validUsername(s string) bool {
	return gitutil.ValidateName(s)
}
