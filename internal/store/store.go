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
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"uniqueIndex"`
	Repository    string
	LatestBackup  time.Time
	SnapshotCount int
	Health        bool
	StatusMessage string
	CheckedAt     time.Time
	Files         []SnapshotFile `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StatusData struct {
	Name          string
	Repository    string
	LatestBackup  time.Time
	SnapshotCount int
	Health        bool
	StatusMessage string
	CheckedAt     time.Time
	Files         []SnapshotFileData
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
	status.SnapshotCount = data.SnapshotCount
	status.Health = data.Health
	status.StatusMessage = data.StatusMessage
	status.CheckedAt = data.CheckedAt

	if err := tx.Save(&status).Error; err != nil {
		return err
	}

	if err := tx.Where("backup_status_id = ?", status.ID).Delete(&SnapshotFile{}).Error; err != nil {
		return err
	}

	files := make([]SnapshotFile, 0, len(data.Files))
	for _, file := range data.Files {
		files = append(files, SnapshotFile{
			BackupStatusID: status.ID,
			Path:           file.Path,
			Name:           file.Name,
			Type:           file.Type,
			Size:           file.Size,
		})
	}

	if len(files) > 0 {
		if err := tx.Create(&files).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) ListStatuses(ctx context.Context) ([]BackupStatus, error) {
	var statuses []BackupStatus
	err := s.db.WithContext(ctx).
		Preload("Files").
		Order("updated_at desc").
		Find(&statuses).Error
	return statuses, err
}

func (s *Store) GetStatus(ctx context.Context, name string) (BackupStatus, error) {
	var status BackupStatus
	err := s.db.WithContext(ctx).
		Preload("Files").
		Where("name = ?", name).
		First(&status).Error
	return status, err
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
