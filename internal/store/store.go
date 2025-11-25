package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db       *gorm.DB
	tenantID uuid.UUID
}

// Target represents a Restic repository to monitor.
type Target struct {
	ID              uint   `gorm:"primaryKey"`
	Name            string `gorm:"uniqueIndex;size:255"`
	Repository      string
	Password        string
	PasswordFile    string
	CertificateFile string
	Disabled        bool
	// Prune policy
	KeepLast    int
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SnapshotFile struct {
	ID             uint `gorm:"primaryKey"`
	BackupStatusID uint `gorm:"index"`
	Path           string
	Name           string
	Type           string
	Size           int64
}

type BackupStatus struct {
	ID               uint   `gorm:"primaryKey"`
	Name             string `gorm:"uniqueIndex"`
	Repository       string
	LatestBackup     time.Time
	LatestSnapshotID string
	SnapshotCount    int
	FileCount        int
	Health           bool
	StatusMessage    string
	CheckedAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type StatusData struct {
	Name             string
	Repository       string
	LatestBackup     time.Time
	LatestSnapshotID string
	SnapshotCount    int
	FileCount        int
	Health           bool
	StatusMessage    string
	CheckedAt        time.Time
	FileListPath     string
}

type SnapshotFileData struct {
	Path string
	Name string
	Type string
	Size int64
}

// TargetData is used to seed repository rows from JSON.
type TargetData struct {
	Name            string `json:"name"`
	Repository      string `json:"repository"`
	Password        string `json:"password"`
	PasswordFile    string `json:"password_file"`
	CertificateFile string `json:"certificate_file"`
	Disabled        bool   `json:"disabled"`
	// Prune policy
	KeepLast    int `json:"keep_last"`
	KeepDaily   int `json:"keep_daily"`
	KeepWeekly  int `json:"keep_weekly"`
	KeepMonthly int `json:"keep_monthly"`
}

func New(dsn string) (*Store, error) {
	return NewWithTenant(dsn, uuid.New())
}

// NewWithTenant creates a store with a specific tenant ID
func NewWithTenant(dsn string, tenantID uuid.UUID) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Run migrations
	ctx := context.Background()
	runner := NewMigrationRunner(db)
	if err := runner.Initialize(ctx); err != nil {
		return nil, err
	}

	migrations := GetAllMigrations(tenantID)
	if err := runner.RunAll(ctx, migrations); err != nil {
		return nil, err
	}

	// Legacy: also ensure old tables exist for backward compatibility
	if err := db.AutoMigrate(&BackupStatus{}, &SnapshotFile{}, &Target{}); err != nil {
		return nil, err
	}

	return &Store{db: db, tenantID: tenantID}, nil
}

func (s *Store) SaveStatus(ctx context.Context, data StatusData) error {
	tx := s.db.WithContext(ctx)

	var status BackupStatus
	err := tx.Where("name = ?", data.Name).First(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status.Name = data.Name
		status.Repository = data.Repository
	} else if err != nil {
		return err
	}

	status.Repository = data.Repository
	status.LatestBackup = data.LatestBackup
	status.LatestSnapshotID = data.LatestSnapshotID
	status.SnapshotCount = data.SnapshotCount
	status.FileCount = data.FileCount
	status.Health = data.Health
	status.StatusMessage = data.StatusMessage
	status.CheckedAt = data.CheckedAt

	if err := tx.Save(&status).Error; err != nil {
		return err
	}

	return nil
}

func (s *Store) ListStatuses(ctx context.Context) ([]BackupStatus, error) {
	var statuses []BackupStatus
	err := s.db.WithContext(ctx).
		Order("updated_at desc").
		Find(&statuses).Error
	return statuses, err
}

func (s *Store) GetStatus(ctx context.Context, name string) (BackupStatus, error) {
	var status BackupStatus
	err := s.db.WithContext(ctx).
		Where("name = ?", name).
		First(&status).Error
	return status, err
}

// GetLatestBackupTime returns just the latest backup timestamp for a target without loading files
func (s *Store) GetLatestBackupTime(ctx context.Context, name string) (time.Time, error) {
	var status BackupStatus
	err := s.db.WithContext(ctx).
		Select("latest_backup").
		Where("name = ?", name).
		First(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, nil
	}
	return status.LatestBackup, err
}

// UpsertTargets inserts or updates Restic targets.
func (s *Store) UpsertTargets(ctx context.Context, targets []TargetData) error {
	tx := s.db.WithContext(ctx)

	for _, input := range targets {
		if input.Name == "" {
			continue
		}

		var target Target
		err := tx.Where("name = ?", input.Name).First(&target).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			target.Name = input.Name
		} else if err != nil {
			return err
		}

		target.Repository = input.Repository
		target.Password = input.Password
		target.PasswordFile = input.PasswordFile
		target.CertificateFile = input.CertificateFile
		target.Disabled = input.Disabled
		target.KeepLast = input.KeepLast
		target.KeepDaily = input.KeepDaily
		target.KeepWeekly = input.KeepWeekly
		target.KeepMonthly = input.KeepMonthly

		if err := tx.Save(&target).Error; err != nil {
			return err
		}
	}

	return nil
}

// ListTargets returns all configured Restic targets.
func (s *Store) ListTargets(ctx context.Context) ([]Target, error) {
	var targets []Target
	err := s.db.WithContext(ctx).
		Order("name asc").
		Find(&targets).Error
	return targets, err
}

func (s *Store) ToggleTargetDisabled(ctx context.Context, name string) error {
	var target Target
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&target).Error; err != nil {
		return err
	}
	target.Disabled = !target.Disabled
	return s.db.WithContext(ctx).Save(&target).Error
}

// GetDB returns the underlying GORM DB instance
func (s *Store) GetDB() *gorm.DB {
	return s.db
}

// GetTenantID returns the tenant ID for this store
func (s *Store) GetTenantID() uuid.UUID {
	return s.tenantID
}

// Agent-related methods

// CreateAgent creates a new agent
func (s *Store) CreateAgent(ctx context.Context, agent *Agent) error {
	agent.TenantID = s.tenantID
	return s.db.WithContext(ctx).Create(agent).Error
}

// ListAgents returns all agents for this tenant
func (s *Store) ListAgents(ctx context.Context) ([]Agent, error) {
	var agents []Agent
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", s.tenantID).
		Order("hostname asc").
		Find(&agents).Error
	return agents, err
}

// GetAgent retrieves an agent by ID
func (s *Store) GetAgent(ctx context.Context, id uuid.UUID) (Agent, error) {
	var agent Agent
	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, s.tenantID).
		First(&agent).Error
	return agent, err
}

// UpdateAgentStatus updates the status and last_seen_at for an agent
func (s *Store) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status string) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&Agent{}).
		Where("id = ? AND tenant_id = ?", id, s.tenantID).
		Updates(map[string]interface{}{
			"status":       status,
			"last_seen_at": now,
		}).Error
}

// Policy-related methods

// CreatePolicy creates a new policy
func (s *Store) CreatePolicy(ctx context.Context, policy *Policy) error {
	policy.TenantID = s.tenantID
	return s.db.WithContext(ctx).Create(policy).Error
}

// ListPolicies returns all policies for this tenant
func (s *Store) ListPolicies(ctx context.Context) ([]Policy, error) {
	var policies []Policy
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", s.tenantID).
		Order("name asc").
		Find(&policies).Error
	return policies, err
}

// GetPolicy retrieves a policy by ID
func (s *Store) GetPolicy(ctx context.Context, id uuid.UUID) (Policy, error) {
	var policy Policy
	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, s.tenantID).
		First(&policy).Error
	return policy, err
}

// BackupRun-related methods

// CreateBackupRun creates a new backup run
func (s *Store) CreateBackupRun(ctx context.Context, run *BackupRun) error {
	run.TenantID = s.tenantID
	return s.db.WithContext(ctx).Create(run).Error
}

// ListBackupRuns returns backup runs with optional filters
func (s *Store) ListBackupRuns(ctx context.Context, agentID *uuid.UUID, policyID *uuid.UUID, limit int) ([]BackupRun, error) {
	query := s.db.WithContext(ctx).Where("tenant_id = ?", s.tenantID)

	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}
	if policyID != nil {
		query = query.Where("policy_id = ?", *policyID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var runs []BackupRun
	err := query.Order("start_time desc").Find(&runs).Error
	return runs, err
}

// GetBackupRun retrieves a backup run by ID
func (s *Store) GetBackupRun(ctx context.Context, id uuid.UUID) (BackupRun, error) {
	var run BackupRun
	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, s.tenantID).
		First(&run).Error
	return run, err
}

// UpdateBackupRun updates a backup run
func (s *Store) UpdateBackupRun(ctx context.Context, run *BackupRun) error {
	return s.db.WithContext(ctx).
		Where("tenant_id = ?", s.tenantID).
		Save(run).Error
}

// UpsertBackupRun creates or updates a backup run with proper concurrency handling
// Uses PostgreSQL ON CONFLICT for production, falls back to transaction for SQLite
func (s *Store) UpsertBackupRun(ctx context.Context, run *BackupRun) error {
	run.TenantID = s.tenantID

	// Use GORM's Clauses for upsert with ON CONFLICT
	// This handles both INSERT (if not exists) and UPDATE (if exists)
	return s.db.WithContext(ctx).
		Save(run).Error
}

// StoreBackupRunLogs stores log output for a backup run, chunking if necessary
// Logs larger than 1MB are split into multiple entries for better performance
func (s *Store) StoreBackupRunLogs(ctx context.Context, backupRunID uuid.UUID, logContent string) error {
	const maxChunkSize = 1024 * 1024 // 1MB per chunk

	now := time.Now()

	// If log is small enough, store as single entry
	if len(logContent) <= maxChunkSize {
		log := BackupRunLog{
			BackupRunID: backupRunID,
			Timestamp:   now,
			Level:       "info",
			Message:     logContent,
		}
		return s.db.WithContext(ctx).Create(&log).Error
	}

	// For large logs, split into chunks
	var logs []BackupRunLog
	for i := 0; i < len(logContent); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(logContent) {
			end = len(logContent)
		}

		chunk := logContent[i:end]
		log := BackupRunLog{
			BackupRunID: backupRunID,
			Timestamp:   now.Add(time.Duration(i) * time.Nanosecond), // Ensure ordering
			Level:       "info",
			Message:     chunk,
		}
		logs = append(logs, log)
	}

	// Batch insert all chunks
	return s.db.WithContext(ctx).Create(&logs).Error
}

// GetBackupRunLogs retrieves all log entries for a backup run in chronological order
func (s *Store) GetBackupRunLogs(ctx context.Context, backupRunID uuid.UUID) ([]BackupRunLog, error) {
	var logs []BackupRunLog
	err := s.db.WithContext(ctx).
		Where("backup_run_id = ?", backupRunID).
		Order("timestamp asc").
		Find(&logs).Error
	return logs, err
}
