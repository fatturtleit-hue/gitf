package models

import "time"

// Workflow represents a CI workflow definition stored in the repo.
type Workflow struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RepoID    uint      `json:"repo_id" gorm:"not null;index"`
	Path      string    `json:"path" gorm:"not null"`      // e.g. ".gitf/workflows/build.yml"
	Name      string    `json:"name" gorm:"not null"`
	TriggerOn string    `json:"trigger_on" gorm:"not null"` // push, pull_request, manual, schedule
	TriggerRef string   `json:"trigger_ref"`                 // branch/tag filter
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Workflow) TableName() string { return "workflows" }

type WorkflowDTO struct {
	ID        uint   `json:"id"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	TriggerOn string `json:"trigger_on"`
	TriggerRef string `json:"trigger_ref"`
}

func (w *Workflow) DTO() WorkflowDTO {
	return WorkflowDTO{ID: w.ID, Path: w.Path, Name: w.Name, TriggerOn: w.TriggerOn, TriggerRef: w.TriggerRef}
}

// WorkflowRun represents a single execution of a workflow.
type WorkflowRun struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	WorkflowID     uint       `json:"workflow_id" gorm:"not null;index"`
	RepoID         uint       `json:"repo_id" gorm:"not null;index"`
	Status         string     `json:"status" gorm:"not null;default:queued"` // queued, dispatched, running, success, failed, cancelled
	TriggerEvent   string     `json:"trigger_event"`
	TriggerPayload string     `json:"trigger_payload" gorm:"type:text"`
	HeadSHA        string     `json:"head_sha"`
	Branch         string     `json:"branch"`
	RunnerID       *uint      `json:"runner_id"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (WorkflowRun) TableName() string { return "workflow_runs" }

// WorkflowRunLog stores chunked log output from a workflow run.
type WorkflowRunLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RunID     uint      `json:"run_id" gorm:"not null;index"`
	Chunk     int       `json:"chunk" gorm:"not null"`
	Content   string    `json:"content" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

func (WorkflowRunLog) TableName() string { return "workflow_run_logs" }

// Runner represents a registered external CI runner.
type Runner struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	TokenHash string    `json:"-" gorm:"uniqueIndex;not null"`
	Status    string    `json:"status" gorm:"default:idle"` // idle, busy, offline
	LastSeenAt *time.Time `json:"last_seen_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (Runner) TableName() string { return "runners" }
