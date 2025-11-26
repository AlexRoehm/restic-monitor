package store

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Agent represents a backup agent running on a target machine
type Agent struct {
	ID               uuid.UUID  `gorm:"primaryKey" json:"id"`
	TenantID         uuid.UUID  `gorm:"not null;index" json:"tenant_id"`
	Hostname         string     `gorm:"type:varchar(255);not null" json:"hostname"`
	OS               string     `gorm:"type:varchar(50);not null" json:"os"`   // linux, windows, darwin
	Arch             string     `gorm:"type:varchar(50);not null" json:"arch"` // amd64, arm64
	Version          string     `gorm:"type:varchar(50);not null" json:"version"`
	Status           string     `gorm:"type:varchar(50);not null" json:"status"` // online, offline, error
	LastSeenAt       *time.Time `gorm:"index" json:"last_seen_at,omitempty"`
	LastBackupStatus string     `gorm:"type:varchar(50)" json:"last_backup_status,omitempty"` // success, failure, none, running
	UptimeSeconds    *int64     `json:"uptime_seconds,omitempty"`
	FreeDisk         JSONB      `gorm:"serializer:json" json:"free_disk,omitempty"` // Array of {mountPath, freeBytes, totalBytes}
	Metadata         JSONB      `gorm:"serializer:json" json:"metadata,omitempty"`
	// Concurrency and quota settings (EPIC 15)
	MaxConcurrentTasks   *int `gorm:"default:3" json:"max_concurrent_tasks,omitempty"`
	MaxConcurrentBackups *int `gorm:"default:1" json:"max_concurrent_backups,omitempty"`
	MaxConcurrentChecks  *int `gorm:"default:1" json:"max_concurrent_checks,omitempty"`
	MaxConcurrentPrunes  *int `gorm:"default:1" json:"max_concurrent_prunes,omitempty"`
	CPUQuotaPercent      *int `gorm:"default:50" json:"cpu_quota_percent,omitempty"`
	BandwidthLimitMbps   *int `json:"bandwidth_limit_mbps,omitempty"`
	// Backoff state tracking (EPIC 15 Phase 6)
	TasksInBackoff  *int       `gorm:"default:0" json:"tasks_in_backoff,omitempty"`
	EarliestRetryAt *time.Time `gorm:"index" json:"earliest_retry_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName specifies the table name for Agent
func (Agent) TableName() string {
	return "agents"
}

// BeforeCreate hook to generate UUID if not set
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// Policy represents a backup policy with schedule and retention rules
type Policy struct {
	ID                 uuid.UUID `gorm:"primaryKey" json:"id"`
	TenantID           uuid.UUID `gorm:"not null;index;uniqueIndex:idx_policy_name_tenant" json:"tenant_id"`
	Name               string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_policy_name_tenant" json:"name"`
	Description        *string   `gorm:"type:varchar(500)" json:"description,omitempty"`
	Schedule           string    `gorm:"type:varchar(255);not null" json:"schedule"`        // backup schedule (cron/interval)
	CheckSchedule      *string   `gorm:"type:varchar(255)" json:"check_schedule,omitempty"` // check schedule (optional)
	PruneSchedule      *string   `gorm:"type:varchar(255)" json:"prune_schedule,omitempty"` // prune schedule (optional)
	IncludePaths       JSONB     `gorm:"serializer:json;not null" json:"include_paths"`
	ExcludePaths       JSONB     `gorm:"serializer:json" json:"exclude_paths,omitempty"`
	RepositoryURL      string    `gorm:"type:text;not null" json:"repository_url"`
	RepositoryType     string    `gorm:"type:varchar(50);not null" json:"repository_type"` // s3, sftp, local, rest
	RepositoryConfig   JSONB     `gorm:"serializer:json" json:"repository_config,omitempty"`
	RetentionRules     JSONB     `gorm:"serializer:json;not null" json:"retention_rules"`
	BandwidthLimitKBps *int      `json:"bandwidth_limit_kbps,omitempty"`
	ParallelFiles      *int      `json:"parallel_files,omitempty"`
	MaxRetries         *int      `gorm:"default:3" json:"max_retries,omitempty"` // EPIC 15 Phase 5
	Enabled            bool      `gorm:"not null" json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TableName specifies the table name for Policy
func (Policy) TableName() string {
	return "policies"
}

// BeforeCreate hook to generate UUID if not set
func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// AgentPolicyLink represents the many-to-many relationship between agents and policies
type AgentPolicyLink struct {
	AgentID   uuid.UUID `gorm:"type:uuid;not null;primaryKey;index" json:"agent_id"`
	PolicyID  uuid.UUID `gorm:"type:uuid;not null;primaryKey;index" json:"policy_id"`
	Agent     Agent     `gorm:"constraint:OnDelete:CASCADE;foreignKey:AgentID;references:ID"`
	Policy    Policy    `gorm:"constraint:OnDelete:CASCADE;foreignKey:PolicyID;references:ID"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for AgentPolicyLink
func (AgentPolicyLink) TableName() string {
	return "agent_policy_links"
}

// BackupRun represents the execution of a backup task
type BackupRun struct {
	ID                  uuid.UUID  `gorm:"primaryKey" json:"id"`
	TenantID            uuid.UUID  `gorm:"not null;index" json:"tenant_id"`
	AgentID             uuid.UUID  `gorm:"not null;index" json:"agent_id"`
	PolicyID            uuid.UUID  `gorm:"not null;index" json:"policy_id"`
	TargetID            *uuid.UUID `gorm:"index" json:"target_id,omitempty"` // optional, for legacy compatibility
	StartTime           time.Time  `gorm:"not null;index:idx_start_time,sort:desc" json:"start_time"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Status              string     `gorm:"type:varchar(50);not null;index" json:"status"` // created, running, success, failed, cancelled
	ErrorMessage        *string    `gorm:"type:text" json:"error_message,omitempty"`
	FilesNew            *int       `json:"files_new,omitempty"`
	FilesChanged        *int       `json:"files_changed,omitempty"`
	FilesUnmodified     *int       `json:"files_unmodified,omitempty"`
	DirsNew             *int       `json:"dirs_new,omitempty"`
	DirsChanged         *int       `json:"dirs_changed,omitempty"`
	DirsUnmodified      *int       `json:"dirs_unmodified,omitempty"`
	DataAdded           *int64     `json:"data_added,omitempty"` // bytes
	TotalFilesProcessed *int64     `json:"total_files_processed,omitempty"`
	TotalBytesProcessed *int64     `json:"total_bytes_processed,omitempty"`
	DurationSeconds     *float64   `json:"duration_seconds,omitempty"`
	SnapshotID          *string    `gorm:"type:varchar(255);index" json:"snapshot_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// TableName specifies the table name for BackupRun
func (BackupRun) TableName() string {
	return "backup_runs"
}

// BeforeCreate hook to generate UUID if not set
func (b *BackupRun) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// BackupRunLog represents individual log entries for a backup run
type BackupRunLog struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`
	BackupRunID uuid.UUID `gorm:"not null;index" json:"backup_run_id"`
	Timestamp   time.Time `gorm:"not null;index:idx_timestamp,sort:desc" json:"timestamp"`
	Level       string    `gorm:"type:varchar(20);not null;index" json:"level"` // debug, info, warn, error
	Message     string    `gorm:"type:text;not null" json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName specifies the table name for BackupRunLog
func (BackupRunLog) TableName() string {
	return "backup_run_logs"
}

// BeforeCreate hook to generate UUID if not set
func (b *BackupRunLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Task represents a task (backup, check, prune) assigned to an agent
type Task struct {
	ID              uuid.UUID  `gorm:"primaryKey" json:"id"`
	TenantID        uuid.UUID  `gorm:"not null;index" json:"tenant_id"`
	AgentID         uuid.UUID  `gorm:"not null;index" json:"agent_id"`
	PolicyID        uuid.UUID  `gorm:"not null;index" json:"policy_id"`
	TaskType        string     `gorm:"type:varchar(50);not null;index" json:"task_type"` // backup, check, prune
	Status          string     `gorm:"type:varchar(50);not null;index" json:"status"`    // pending, assigned, in-progress, completed, failed
	Repository      string     `gorm:"type:text;not null" json:"repository"`
	IncludePaths    JSONB      `gorm:"serializer:json" json:"include_paths,omitempty"`
	ExcludePaths    JSONB      `gorm:"serializer:json" json:"exclude_paths,omitempty"`
	Retention       JSONB      `gorm:"serializer:json" json:"retention,omitempty"`
	ExecutionParams JSONB      `gorm:"serializer:json" json:"execution_params,omitempty"`
	ScheduledFor    *time.Time `gorm:"index" json:"scheduled_for,omitempty"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ErrorMessage    *string    `gorm:"type:text" json:"error_message,omitempty"`

	// Retry tracking fields (EPIC 15)
	RetryCount        *int       `gorm:"default:0" json:"retry_count,omitempty"`
	MaxRetries        *int       `gorm:"default:3" json:"max_retries,omitempty"`
	NextRetryAt       *time.Time `gorm:"index" json:"next_retry_at,omitempty"`
	LastErrorCategory *string    `gorm:"type:varchar(100)" json:"last_error_category,omitempty"`

	CreatedAt time.Time `gorm:"index" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for Task
func (Task) TableName() string {
	return "tasks"
}

// BeforeCreate hook to generate UUID if not set
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// MigrateModels runs GORM automigration for all models
func MigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&Agent{},
		&Policy{},
		&AgentPolicyLink{},
		&BackupRun{},
		&BackupRunLog{},
		&Task{},
	)
}
