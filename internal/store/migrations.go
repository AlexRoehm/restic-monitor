package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SchemaMigration tracks which migrations have been applied
type SchemaMigration struct {
	Version     string    `gorm:"primaryKey;size:50" json:"version"`
	Description string    `gorm:"size:255" json:"description"`
	AppliedAt   time.Time `gorm:"not null" json:"applied_at"`
}

// TableName specifies the table name for SchemaMigration
func (SchemaMigration) TableName() string {
	return "schema_migrations"
}

// Migration represents a database migration
type Migration struct {
	Version     string
	Description string
	Up          func(tx *gorm.DB) error
}

// MigrationRunner manages database migrations
type MigrationRunner struct {
	db *gorm.DB
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *gorm.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// Initialize creates the schema_migrations table
func (r *MigrationRunner) Initialize(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&SchemaMigration{})
}

// IsApplied checks if a migration has been applied
func (r *MigrationRunner) IsApplied(ctx context.Context, version string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SchemaMigration{}).Where("version = ?", version).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Run executes a single migration if it hasn't been applied
func (r *MigrationRunner) Run(ctx context.Context, migration Migration) error {
	// Check if already applied
	applied, err := r.IsApplied(ctx, migration.Version)
	if err != nil {
		return fmt.Errorf("checking migration status: %w", err)
	}
	if applied {
		return nil // Already applied, skip
	}

	// Run migration in a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Execute migration
		if err := migration.Up(tx); err != nil {
			return fmt.Errorf("executing migration %s: %w", migration.Version, err)
		}

		// Record migration
		record := SchemaMigration{
			Version:     migration.Version,
			Description: migration.Description,
			AppliedAt:   time.Now(),
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("recording migration %s: %w", migration.Version, err)
		}

		return nil
	})
}

// RunAll executes all pending migrations in order
func (r *MigrationRunner) RunAll(ctx context.Context, migrations []Migration) error {
	// Sort by version to ensure order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for _, migration := range migrations {
		if err := r.Run(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

// GetAllMigrations returns all available migrations
func GetAllMigrations(defaultTenantID uuid.UUID) []Migration {
	return []Migration{
		GetMigration001CreateAgentTables(defaultTenantID),
		GetMigration002AddHeartbeatFields(),
	}
}

// GetMigration001CreateAgentTables creates the v1.x schema and migrates data from v0.x
func GetMigration001CreateAgentTables(defaultTenantID uuid.UUID) Migration {
	return Migration{
		Version:     "001",
		Description: "Create agent, policy, and backup run tables; migrate from v0.x schema",
		Up: func(tx *gorm.DB) error {
			// Create new v1 tables
			if err := MigrateModels(tx); err != nil {
				return fmt.Errorf("creating v1 tables: %w", err)
			}

			// Check if legacy Target table exists
			var legacyTableExists bool
			err := tx.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='targets'").Scan(&legacyTableExists).Error
			if err != nil {
				return fmt.Errorf("checking for legacy tables: %w", err)
			}

			if !legacyTableExists {
				// No legacy data to migrate
				return nil
			}

			// Migrate legacy Target data to Agent + Policy
			var targets []Target
			if err := tx.Find(&targets).Error; err != nil {
				return fmt.Errorf("loading legacy targets: %w", err)
			}

			for _, target := range targets {
				// Create an agent for this target
				agent := Agent{
					TenantID: defaultTenantID,
					Hostname: target.Name,
					OS:       "unknown", // Not available in v0 schema
					Arch:     "unknown",
					Version:  "legacy",
					Status:   "offline",
					Metadata: JSONB{
						"migrated_from": "v0_target",
						"target_id":     target.ID,
					},
				}

				if err := tx.Create(&agent).Error; err != nil {
					return fmt.Errorf("creating agent from target %s: %w", target.Name, err)
				}

				// Create a policy from the target configuration
				policy := Policy{
					TenantID:       defaultTenantID,
					Name:           fmt.Sprintf("%s Policy", target.Name),
					Schedule:       "0 2 * * *", // Default daily at 2am
					IncludePaths:   JSONB{"paths": []string{"/"}},
					RepositoryURL:  target.Repository,
					RepositoryType: detectRepositoryType(target.Repository),
					RetentionRules: JSONB{
						"keep_last":    target.KeepLast,
						"keep_daily":   target.KeepDaily,
						"keep_weekly":  target.KeepWeekly,
						"keep_monthly": target.KeepMonthly,
					},
					Enabled: !target.Disabled,
				}

				if err := tx.Create(&policy).Error; err != nil {
					return fmt.Errorf("creating policy from target %s: %w", target.Name, err)
				}

				// Link agent and policy
				link := AgentPolicyLink{
					AgentID:  agent.ID,
					PolicyID: policy.ID,
				}

				if err := tx.Create(&link).Error; err != nil {
					return fmt.Errorf("linking agent and policy: %w", err)
				}
			}

			return nil
		},
	}
}

// detectRepositoryType attempts to detect the repository type from the URL
func detectRepositoryType(repoURL string) string {
	if len(repoURL) == 0 {
		return "local"
	}

	// Simple detection based on prefix
	switch {
	case len(repoURL) >= 3 && repoURL[:3] == "s3:":
		return "s3"
	case len(repoURL) >= 5 && repoURL[:5] == "sftp:":
		return "sftp"
	case len(repoURL) >= 5 && repoURL[:5] == "rest:":
		return "rest"
	case len(repoURL) >= 7 && repoURL[:7] == "http://", len(repoURL) >= 8 && repoURL[:8] == "https://":
		return "rest"
	default:
		return "local"
	}
}

// GetMigration002AddHeartbeatFields adds heartbeat-related fields to agents table
func GetMigration002AddHeartbeatFields() Migration {
	return Migration{
		Version:     "002",
		Description: "Add heartbeat fields to agents table (last_backup_status, uptime_seconds, free_disk)",
		Up: func(tx *gorm.DB) error {
			// Add new columns to agents table
			// GORM AutoMigrate handles adding new columns gracefully
			if err := tx.AutoMigrate(&Agent{}); err != nil {
				return fmt.Errorf("adding heartbeat fields to agents: %w", err)
			}
			return nil
		},
	}
}
