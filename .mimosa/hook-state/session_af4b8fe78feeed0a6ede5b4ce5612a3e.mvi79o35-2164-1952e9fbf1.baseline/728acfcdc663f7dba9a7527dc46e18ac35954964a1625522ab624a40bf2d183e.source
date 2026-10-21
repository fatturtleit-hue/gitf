package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"gitf/internal/models"

	"gorm.io/gorm"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

// TokenStore abstracts PAT persistence.
type TokenStore interface {
	Create(userID uint, name, scopes string) (plainToken string, token *models.PersonalAccessToken, err error)
	List(userID uint) ([]models.PersonalAccessToken, error)
	Delete(userID, tokenID uint) error
	Lookup(plainToken string) (*models.PersonalAccessToken, error)
}

// GormTokenStore implements TokenStore using GORM.
type GormTokenStore struct{ DB *gorm.DB }

func NewTokenStore(db *gorm.DB) TokenStore { return &GormTokenStore{DB: db} }

func (s *GormTokenStore) Create(userID uint, name, scopes string) (string, *models.PersonalAccessToken, error) {
	// Generate random token bytes.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	plain := "gitf_" + hex.EncodeToString(b)
	hash := sha256Hash(plain)

	t := models.PersonalAccessToken{
		UserID:    userID,
		Name:      name,
		TokenHash: hash,
		Prefix:    plain[:13], // "gitf_" + first 8 hex chars
		Scopes:    scopes,
	}
	if err := s.DB.Create(&t).Error; err != nil {
		return "", nil, fmt.Errorf("create token: %w", err)
	}
	return plain, &t, nil
}

func (s *GormTokenStore) List(userID uint) ([]models.PersonalAccessToken, error) {
	var tokens []models.PersonalAccessToken
	if err := s.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *GormTokenStore) Delete(userID, tokenID uint) error {
	res := s.DB.Where("user_id = ? AND id = ?", userID, tokenID).Delete(&models.PersonalAccessToken{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (s *GormTokenStore) Lookup(plainToken string) (*models.PersonalAccessToken, error) {
	hash := sha256Hash(plainToken)
	var t models.PersonalAccessToken
	if err := s.DB.Where("token_hash = ?", hash).First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	// Update last used.
	s.DB.Model(&t).Update("last_used_at", gorm.Expr("datetime('now')"))
	return &t, nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
