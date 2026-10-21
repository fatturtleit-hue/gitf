package models

// BranchProtection defines protection rules for branches matching a glob pattern.
type BranchProtection struct {
	ID                   uint   `json:"id" gorm:"primaryKey"`
	RepoID               uint   `json:"repo_id" gorm:"not null;index"`
	Pattern              string `json:"pattern" gorm:"not null"` // glob, e.g. "main", "release/*"
	RequireApprovals     int    `json:"require_approvals" gorm:"default:0"`
	AllowForcePush       bool   `json:"allow_force_push"`
	AllowDelete          bool   `json:"allow_delete"`
	RequiredStatusChecks string `json:"required_status_checks"` // comma-separated check names
	EnforceAdmins        bool   `json:"enforce_admins"`
}

func (BranchProtection) TableName() string { return "branch_protections" }

type BranchProtectionDTO struct {
	ID                   uint   `json:"id"`
	RepoID               uint   `json:"repo_id"`
	Pattern              string `json:"pattern"`
	RequireApprovals     int    `json:"require_approvals"`
	AllowForcePush       bool   `json:"allow_force_push"`
	AllowDelete          bool   `json:"allow_delete"`
	RequiredStatusChecks string `json:"required_status_checks"`
	EnforceAdmins        bool   `json:"enforce_admins"`
}

func (bp *BranchProtection) DTO() BranchProtectionDTO {
	return BranchProtectionDTO{
		ID:                   bp.ID,
		RepoID:               bp.RepoID,
		Pattern:              bp.Pattern,
		RequireApprovals:     bp.RequireApprovals,
		AllowForcePush:       bp.AllowForcePush,
		AllowDelete:          bp.AllowDelete,
		RequiredStatusChecks: bp.RequiredStatusChecks,
		EnforceAdmins:        bp.EnforceAdmins,
	}
}
