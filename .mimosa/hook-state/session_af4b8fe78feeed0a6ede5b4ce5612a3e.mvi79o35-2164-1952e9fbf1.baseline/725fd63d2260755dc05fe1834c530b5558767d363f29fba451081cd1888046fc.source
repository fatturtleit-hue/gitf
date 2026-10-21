package models

import "time"

// Package represents a published package in the registry.
type Package struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OwnerType   string    `json:"owner_type" gorm:"not null;index;uniqueIndex:idx_pkg_owner_type_name"` // user, org
	OwnerID     uint      `json:"owner_id" gorm:"not null;uniqueIndex:idx_pkg_owner_type_name"`
	Type        string    `json:"type" gorm:"not null;uniqueIndex:idx_pkg_owner_type_name"` // docker, npm, pypi, maven, nuget, helm
	Name        string    `json:"name" gorm:"not null;uniqueIndex:idx_pkg_owner_type_name"`
	LowerName   string    `json:"lower_name" gorm:"not null"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Package) TableName() string { return "packages" }

type PackageVersion struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	PackageID   uint      `json:"package_id" gorm:"not null;index;uniqueIndex:idx_pkg_ver"`
	Version     string    `json:"version" gorm:"not null;uniqueIndex:idx_pkg_ver"`
	Metadata    string    `json:"metadata" gorm:"type:text"` // JSON
	StoragePath string    `json:"-" gorm:"not null"`
	Size        int64     `json:"size"`
	AuthorID    uint      `json:"author_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (PackageVersion) TableName() string { return "package_versions" }

type PackageFile struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	VersionID   uint      `json:"version_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	StoragePath string    `json:"-" gorm:"not null"`
	Digest      string    `json:"digest"` // SHA256
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

func (PackageFile) TableName() string { return "package_files" }
