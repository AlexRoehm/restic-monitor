package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
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
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&BackupStatus{}, &SnapshotFile{}, &Target{}); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
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
