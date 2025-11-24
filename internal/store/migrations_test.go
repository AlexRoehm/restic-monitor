package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Helper function for int pointer (duplicated from models_test.go)
func intPtrMigration(i int) *int {
	return &i
}

// TestMigrationRunner tests the migration runner (TDD)
func TestMigrationRunner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Initialize schema_migrations table", func(t *testing.T) {
		runner := store.NewMigrationRunner(db)
		err := runner.Initialize(ctx)
		require.NoError(t, err)

		// Verify table exists
		var count int64
		err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Run single migration", func(t *testing.T) {
		// Reinitialize with fresh DB
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		runner := store.NewMigrationRunner(db)
		runner.Initialize(ctx)

		executed := false
		migration := store.Migration{
			Version:     "001",
			Description: "test migration",
			Up: func(tx *gorm.DB) error {
				executed = true
				return tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)").Error
			},
		}

		err := runner.Run(ctx, migration)
		require.NoError(t, err)
		assert.True(t, executed, "Migration should have been executed")

		// Verify migration was recorded
		var record store.SchemaMigration
		err = db.Where("version = ?", "001").First(&record).Error
		require.NoError(t, err)
		assert.Equal(t, "001", record.Version)
		assert.Equal(t, "test migration", record.Description)
		assert.True(t, record.AppliedAt.After(time.Now().Add(-5*time.Second)))
	})

	t.Run("Skip already applied migration", func(t *testing.T) {
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		runner := store.NewMigrationRunner(db)
		runner.Initialize(ctx)

		executed := 0
		migration := store.Migration{
			Version:     "001",
			Description: "test migration",
			Up: func(tx *gorm.DB) error {
				executed++
				return nil
			},
		}

		// Run once
		err := runner.Run(ctx, migration)
		require.NoError(t, err)
		assert.Equal(t, 1, executed)

		// Run again - should be skipped
		err = runner.Run(ctx, migration)
		require.NoError(t, err)
		assert.Equal(t, 1, executed, "Migration should not execute twice")
	})

	t.Run("Run multiple migrations in order", func(t *testing.T) {
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		runner := store.NewMigrationRunner(db)
		runner.Initialize(ctx)

		order := []string{}
		migrations := []store.Migration{
			{
				Version:     "001",
				Description: "first",
				Up: func(tx *gorm.DB) error {
					order = append(order, "001")
					return nil
				},
			},
			{
				Version:     "002",
				Description: "second",
				Up: func(tx *gorm.DB) error {
					order = append(order, "002")
					return nil
				},
			},
			{
				Version:     "003",
				Description: "third",
				Up: func(tx *gorm.DB) error {
					order = append(order, "003")
					return nil
				},
			},
		}

		err := runner.RunAll(ctx, migrations)
		require.NoError(t, err)
		assert.Equal(t, []string{"001", "002", "003"}, order)
	})
}

// TestMigrationV0ToV1 tests the actual v0.x to v1.x migration (TDD)
func TestMigrationV0ToV1(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Migrate v0 schema to v1 with data preservation", func(t *testing.T) {
		// Create v0 schema (legacy Target, BackupStatus tables)
		err := db.AutoMigrate(&store.Target{}, &store.BackupStatus{}, &store.SnapshotFile{})
		require.NoError(t, err)

		// Insert legacy data
		tenantID := uuid.New()
		legacyTarget := store.Target{
			Name:        "test-backup",
			Repository:  "s3:bucket/path",
			Password:    "secret",
			Disabled:    false,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 12,
		}
		err = db.Create(&legacyTarget).Error
		require.NoError(t, err)

		// Run migration
		runner := store.NewMigrationRunner(db)
		err = runner.Initialize(ctx)
		require.NoError(t, err)

		migration := store.GetMigration001CreateAgentTables(tenantID)
		err = runner.Run(ctx, migration)
		require.NoError(t, err)

		// Verify new tables exist
		tables := []string{"agents", "policies", "agent_policy_links", "backup_runs", "backup_run_logs"}
		for _, table := range tables {
			var count int64
			err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
			require.NoError(t, err)
			assert.Equal(t, int64(1), count, "Table %s should exist", table)
		}

		// Verify data was migrated
		var policies []store.Policy
		err = db.Find(&policies).Error
		require.NoError(t, err)
		assert.Greater(t, len(policies), 0, "Should have migrated policies")

		// Verify retention rules were preserved
		if len(policies) > 0 {
			policy := policies[0]
			assert.Equal(t, tenantID, policy.TenantID)
			assert.NotEmpty(t, policy.Name)
			assert.Equal(t, float64(7), policy.RetentionRules["keep_daily"])
			assert.Equal(t, float64(4), policy.RetentionRules["keep_weekly"])
			assert.Equal(t, float64(12), policy.RetentionRules["keep_monthly"])
		}
	})

	t.Run("Handle empty v0 database", func(t *testing.T) {
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})

		// Create empty v0 schema
		err := db.AutoMigrate(&store.Target{}, &store.BackupStatus{})
		require.NoError(t, err)

		// Run migration
		runner := store.NewMigrationRunner(db)
		err = runner.Initialize(ctx)
		require.NoError(t, err)

		tenantID := uuid.New()
		migration := store.GetMigration001CreateAgentTables(tenantID)
		err = runner.Run(ctx, migration)
		require.NoError(t, err)

		// Should succeed even with no data
		var count int64
		err = db.Model(&store.Agent{}).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

// TestGetAllMigrations tests that all migrations are available
func TestGetAllMigrations(t *testing.T) {
	tenantID := uuid.New()
	migrations := store.GetAllMigrations(tenantID)

	require.NotEmpty(t, migrations, "Should have at least one migration")

	// Verify first migration
	assert.Equal(t, "001", migrations[0].Version)
	assert.Contains(t, migrations[0].Description, "agent")
	assert.NotNil(t, migrations[0].Up)

	// Verify migrations are in order
	for i := 1; i < len(migrations); i++ {
		assert.Greater(t, migrations[i].Version, migrations[i-1].Version,
			"Migrations should be in version order")
	}

	// Verify we have all expected migrations
	assert.GreaterOrEqual(t, len(migrations), 3, "Should have at least 3 migrations")
}

// TestMigration003PolicyFields tests the policy fields migration
func TestMigration003PolicyFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Add new policy fields", func(t *testing.T) {
		// Create v1 schema first (migration 001)
		tenantID := uuid.New()
		runner := store.NewMigrationRunner(db)
		err := runner.Initialize(ctx)
		require.NoError(t, err)

		migration001 := store.GetMigration001CreateAgentTables(tenantID)
		err = runner.Run(ctx, migration001)
		require.NoError(t, err)

		// Create a policy with old schema (before migration 003)
		oldPolicy := store.Policy{
			TenantID:       tenantID,
			Name:           "old-policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err = db.Create(&oldPolicy).Error
		require.NoError(t, err)

		// Run migration 003
		migration003 := store.GetMigration003AddPolicyFields()
		err = runner.Run(ctx, migration003)
		require.NoError(t, err)

		// Create a new policy with new fields
		description := "Test policy with bandwidth limits"
		newPolicy := store.Policy{
			TenantID:           tenantID,
			Name:               "new-policy",
			Description:        &description,
			Schedule:           "0 3 * * *",
			IncludePaths:       store.JSONB{"paths": []string{"/var"}},
			RepositoryURL:      "s3:bucket/new",
			RepositoryType:     "s3",
			RetentionRules:     store.JSONB{"keep_daily": 14},
			BandwidthLimitKBps: intPtrMigration(5120),
			ParallelFiles:      intPtrMigration(2),
			Enabled:            true,
		}
		err = db.Create(&newPolicy).Error
		require.NoError(t, err)

		// Verify new fields are persisted
		var retrieved store.Policy
		err = db.First(&retrieved, "id = ?", newPolicy.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Description)
		assert.Equal(t, description, *retrieved.Description)
		assert.NotNil(t, retrieved.BandwidthLimitKBps)
		assert.Equal(t, 5120, *retrieved.BandwidthLimitKBps)
		assert.NotNil(t, retrieved.ParallelFiles)
		assert.Equal(t, 2, *retrieved.ParallelFiles)

		// Verify old policy still exists
		var oldRetrieved store.Policy
		err = db.First(&oldRetrieved, "id = ?", oldPolicy.ID).Error
		require.NoError(t, err)
		assert.Nil(t, oldRetrieved.Description)
		assert.Nil(t, oldRetrieved.BandwidthLimitKBps)
		assert.Nil(t, oldRetrieved.ParallelFiles)
	})

	t.Run("Name uniqueness constraint", func(t *testing.T) {
		// Create a fresh database for this test
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})

		ctx := context.Background()
		tenantID := uuid.New()
		runner := store.NewMigrationRunner(db)
		err := runner.Initialize(ctx)
		require.NoError(t, err)

		// Run migrations
		migrations := []store.Migration{
			store.GetMigration001CreateAgentTables(tenantID),
			store.GetMigration003AddPolicyFields(),
		}
		err = runner.RunAll(ctx, migrations)
		require.NoError(t, err)

		// Create first policy
		policy1 := store.Policy{
			TenantID:       tenantID,
			Name:           "unique-name",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err = db.Create(&policy1).Error
		require.NoError(t, err)

		// Try to create second policy with same name
		policy2 := store.Policy{
			TenantID:       tenantID,
			Name:           "unique-name",
			Schedule:       "0 3 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/other"}},
			RepositoryURL:  "s3:bucket/other",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err = db.Create(&policy2).Error
		assert.Error(t, err, "Should fail due to unique constraint")
	})
}
