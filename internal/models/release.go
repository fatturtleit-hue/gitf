package models

import "time"

// Release represents a published software release tied to a Git tag.
type Release struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	RepoID          uint      `json:"repo_id" gorm:"not null;uniqueIndex:idx_repo_tag,priority:1"`
	TagName         string    `json:"tag_name" gorm:"not null;uniqueIndex:idx_repo_tag,priority:2"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Body            string    `json:"body"`
	IsDraft         bool      `json:"is_draft"`
	IsPrerelease    bool      `json:"is_prerelease"`
	AuthorID        uint      `json:"author_id" gorm:"not null"`
	Author          User      `json:"author" gorm:"constraint:OnDelete:CASCADE;"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Release) TableName() string { return "releases" }

type ReleaseDTO struct {
	ID              uint   `json:"id"`
	RepoID          uint   `json:"repo_id"`
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	IsDraft         bool   `json:"is_draft"`
	IsPrerelease    bool   `json:"is_prerelease"`
	AuthorID        uint   `json:"author_id"`
	AuthorUsername  string `json:"author_username"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func (r *Release) DTO() ReleaseDTO {
	return ReleaseDTO{
		ID:              r.ID,
		RepoID:          r.RepoID,
		TagName:         r.TagName,
		TargetCommitish: r.TargetCommitish,
		Name:            r.Name,
		Body:            r.Body,
		IsDraft:         r.IsDraft,
		IsPrerelease:    r.IsPrerelease,
		AuthorID:        r.AuthorID,
		AuthorUsername:  r.Author.Username,
		CreatedAt:       r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ReleaseAsset represents a file attached to a release.
type ReleaseAsset struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ReleaseID   uint      `json:"release_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	StoragePath string    `json:"-" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ReleaseAsset) TableName() string { return "release_assets" }

type ReleaseAssetDTO struct {
	ID          uint   `json:"id"`
	ReleaseID   uint   `json:"release_id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	CreatedAt   string `json:"created_at"`
}

func (a *ReleaseAsset) DTO() ReleaseAssetDTO {
	return ReleaseAssetDTO{
		ID:          a.ID,
		ReleaseID:   a.ReleaseID,
		Name:        a.Name,
		Size:        a.Size,
		ContentType: a.ContentType,
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
