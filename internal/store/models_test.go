package store_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
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

// TestModelsCompile ensures all models compile (TDD: fail early)
func TestModelsCompile(t *testing.T) {
	// If this test compiles and runs, models are syntactically correct
	_ = store.Agent{}
	_ = store.Policy{}
	_ = store.AgentPolicyLink{}
	_ = store.BackupRun{}
	_ = store.BackupRunLog{}
	assert.True(t, true, "All models compile successfully")
}

// TestModelFieldsExist validates required fields exist using reflection (TDD)
func TestModelFieldsExist(t *testing.T) {
	tests := []struct {
		name           string
		model          interface{}
		requiredFields []string
	}{
		{
			name:  "Agent",
			model: store.Agent{},
			requiredFields: []string{
				"ID", "TenantID", "Hostname", "OS", "Arch", "Version",
				"Status", "LastSeenAt", "Metadata", "CreatedAt", "UpdatedAt",
			},
		},
		{
			name:  "Policy",
			model: store.Policy{},
			requiredFields: []string{
				"ID", "TenantID", "Name", "Description", "Schedule", "IncludePaths",
				"ExcludePaths", "RepositoryURL", "RepositoryType",
				"RepositoryConfig", "RetentionRules", "BandwidthLimitKBps",
				"ParallelFiles", "Enabled", "CreatedAt", "UpdatedAt",
			},
		},
		{
			name:           "AgentPolicyLink",
			model:          store.AgentPolicyLink{},
			requiredFields: []string{"AgentID", "PolicyID", "CreatedAt"},
		},
		{
			name:  "BackupRun",
			model: store.BackupRun{},
			requiredFields: []string{
				"ID", "TenantID", "AgentID", "PolicyID", "StartTime",
				"EndTime", "Status", "ErrorMessage", "FilesNew",
				"FilesChanged", "FilesUnmodified", "DirsNew", "DirsChanged",
				"DirsUnmodified", "DataAdded", "TotalFilesProcessed",
				"TotalBytesProcessed", "DurationSeconds", "SnapshotID",
				"CreatedAt", "UpdatedAt",
			},
		},
		{
			name:  "BackupRunLog",
			model: store.BackupRunLog{},
			requiredFields: []string{
				"ID", "BackupRunID", "Timestamp", "Level", "Message", "CreatedAt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelType := reflect.TypeOf(tt.model)
			for _, fieldName := range tt.requiredFields {
				field, found := modelType.FieldByName(fieldName)
				assert.True(t, found, "Field %s must exist in %s", fieldName, tt.name)

				// Validate JSON tag exists
				if found {
					jsonTag := field.Tag.Get("json")
					assert.NotEmpty(t, jsonTag, "Field %s must have json tag", fieldName)
				}
			}
		})
	}
}

// TestModelSerialization tests JSON serialization and deserialization (TDD)
func TestModelSerialization(t *testing.T) {
	tenantID := uuid.New()
	agentID := uuid.New()
	policyID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name  string
		model interface{}
	}{
		{
			name: "Agent",
			model: &store.Agent{
				ID:         agentID,
				TenantID:   tenantID,
				Hostname:   "web-server-01",
				OS:         "linux",
				Arch:       "amd64",
				Version:    "1.2.3",
				Status:     "online",
				LastSeenAt: &now,
				Metadata: store.JSONB{
					"restic_version": "0.16.2",
					"plugins":        []string{"mysql", "postgres"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Policy",
			model: &store.Policy{
				ID:          policyID,
				TenantID:    tenantID,
				Name:        "Daily Production Backup",
				Description: stringPtr("Daily backup of production data to S3"),
				Schedule:    "0 2 * * *",
				IncludePaths: store.JSONB{
					"paths": []string{"/home", "/etc"},
				},
				ExcludePaths: store.JSONB{
					"patterns": []string{"*.log", ".cache"},
				},
				RepositoryURL:  "s3:s3.amazonaws.com/my-backups",
				RepositoryType: "s3",
				RepositoryConfig: store.JSONB{
					"bucket": "my-backups",
					"region": "us-east-1",
				},
				RetentionRules: store.JSONB{
					"keep_daily":   7,
					"keep_weekly":  4,
					"keep_monthly": 12,
				},
				BandwidthLimitKBps: intPtr(10240),
				ParallelFiles:      intPtr(4),
				Enabled:            true,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
		{
			name: "AgentPolicyLink",
			model: &store.AgentPolicyLink{
				AgentID:   agentID,
				PolicyID:  policyID,
				CreatedAt: now,
			},
		},
		{
			name: "BackupRun",
			model: &store.BackupRun{
				ID:                  uuid.New(),
				TenantID:            tenantID,
				AgentID:             agentID,
				PolicyID:            policyID,
				StartTime:           now,
				EndTime:             &now,
				Status:              "success",
				FilesNew:            intPtr(1234),
				FilesChanged:        intPtr(567),
				FilesUnmodified:     intPtr(89012),
				DataAdded:           int64Ptr(5368709120),
				TotalFilesProcessed: int64Ptr(90813),
				TotalBytesProcessed: int64Ptr(107374182400),
				DurationSeconds:     float64Ptr(930.5),
				SnapshotID:          stringPtr("a1b2c3d4"),
				CreatedAt:           now,
				UpdatedAt:           now,
			},
		},
		{
			name: "BackupRunLog",
			model: &store.BackupRunLog{
				ID:          uuid.New(),
				BackupRunID: uuid.New(),
				Timestamp:   now,
				Level:       "info",
				Message:     "processed 10000 files, 5.2 GB",
				CreatedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			jsonData, err := json.Marshal(tt.model)
			require.NoError(t, err, "Serialization should succeed")
			assert.NotEmpty(t, jsonData, "JSON data should not be empty")

			// Deserialize from JSON
			modelType := reflect.TypeOf(tt.model).Elem()
			newModel := reflect.New(modelType).Interface()
			err = json.Unmarshal(jsonData, newModel)
			require.NoError(t, err, "Deserialization should succeed")

			// Verify round-trip (note: JSON unmarshaling converts arrays to []interface{} and numbers to float64)
			// So we just verify that marshaling works both ways without errors
			jsonData2, err := json.Marshal(newModel)
			require.NoError(t, err, "Re-serialization should succeed")
			assert.NotEmpty(t, jsonData2, "Re-serialized JSON should not be empty")
		})
	}
}

// TestJSONBCustomType tests the JSONB custom type (TDD)
func TestJSONBCustomType(t *testing.T) {
	t.Run("Value - Nil", func(t *testing.T) {
		var j store.JSONB
		value, err := j.Value()
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Value - Valid", func(t *testing.T) {
		j := store.JSONB{"key": "value", "num": 42}
		value, err := j.Value()
		require.NoError(t, err)
		assert.NotNil(t, value)

		// Should be valid JSON
		var decoded map[string]interface{}
		err = json.Unmarshal(value.([]byte), &decoded)
		require.NoError(t, err)
		assert.Equal(t, "value", decoded["key"])
		assert.Equal(t, float64(42), decoded["num"])
	})

	t.Run("Scan - Nil", func(t *testing.T) {
		var j store.JSONB
		err := j.Scan(nil)
		require.NoError(t, err)
		assert.Nil(t, j)
	})

	t.Run("Scan - Valid", func(t *testing.T) {
		jsonBytes := []byte(`{"key":"value","num":42}`)
		var j store.JSONB
		err := j.Scan(jsonBytes)
		require.NoError(t, err)
		assert.Equal(t, "value", j["key"])
		assert.Equal(t, float64(42), j["num"])
	})
}

// TestMigrateModels tests database migration (TDD)
func TestMigrateModels(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Database connection should succeed")

	// Run migrations
	err = store.MigrateModels(db)
	require.NoError(t, err, "Migrations should succeed")

	// Verify tables exist
	tables := []string{
		"agents",
		"policies",
		"agent_policy_links",
		"backup_runs",
		"backup_run_logs",
	}
	for _, table := range tables {
		var count int64
		err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
		require.NoError(t, err, "Table check query should succeed")
		assert.Equal(t, int64(1), count, "Table %s should exist", table)
	}
}

// TestModelCRUD tests basic CRUD operations (TDD)
func TestModelCRUD(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	err = store.MigrateModels(db)
	require.NoError(t, err)

	tenantID := uuid.New()

	t.Run("Agent CRUD", func(t *testing.T) {
		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "test-server",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
			Metadata: store.JSONB{"test": "data"},
		}

		// Create
		err := db.Create(&agent).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, agent.ID)

		// Read
		var retrieved store.Agent
		err = db.First(&retrieved, "id = ?", agent.ID).Error
		require.NoError(t, err)
		assert.Equal(t, agent.Hostname, retrieved.Hostname)
		assert.Equal(t, "data", retrieved.Metadata["test"])

		// Update
		err = db.Model(&agent).Update("status", "offline").Error
		require.NoError(t, err)

		err = db.First(&retrieved, "id = ?", agent.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "offline", retrieved.Status)

		// Delete
		err = db.Delete(&agent).Error
		require.NoError(t, err)

		err = db.First(&retrieved, "id = ?", agent.ID).Error
		assert.Error(t, err) // Should not be found
	})

	t.Run("Policy CRUD", func(t *testing.T) {
		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "Test Policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			ExcludePaths:   store.JSONB{"patterns": []string{"*.tmp"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}

		err := db.Create(&policy).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, policy.ID)

		var retrieved store.Policy
		err = db.First(&retrieved, "id = ?", policy.ID).Error
		require.NoError(t, err)
		assert.Equal(t, policy.Name, retrieved.Name)
		assert.Equal(t, float64(7), retrieved.RetentionRules["keep_daily"])
	})

	t.Run("Policy with Optional Fields", func(t *testing.T) {
		description := "Production backup with bandwidth limits"
		policy := store.Policy{
			TenantID:           tenantID,
			Name:               "Limited Policy",
			Description:        &description,
			Schedule:           "0 3 * * *",
			IncludePaths:       store.JSONB{"paths": []string{"/var/www"}},
			RepositoryURL:      "s3:bucket/limited",
			RepositoryType:     "s3",
			RetentionRules:     store.JSONB{"keep_daily": 14},
			BandwidthLimitKBps: intPtr(5120),
			ParallelFiles:      intPtr(2),
			Enabled:            true,
		}

		err := db.Create(&policy).Error
		require.NoError(t, err)

		var retrieved store.Policy
		err = db.First(&retrieved, "id = ?", policy.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Description)
		assert.Equal(t, description, *retrieved.Description)
		assert.NotNil(t, retrieved.BandwidthLimitKBps)
		assert.Equal(t, 5120, *retrieved.BandwidthLimitKBps)
		assert.NotNil(t, retrieved.ParallelFiles)
		assert.Equal(t, 2, *retrieved.ParallelFiles)
	})

	t.Run("Policy Name Uniqueness", func(t *testing.T) {
		policy1 := store.Policy{
			TenantID:       tenantID,
			Name:           "Unique Policy Name",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err := db.Create(&policy1).Error
		require.NoError(t, err)

		// Attempt to create another policy with the same name and tenant
		policy2 := store.Policy{
			TenantID:       tenantID,
			Name:           "Unique Policy Name",
			Schedule:       "0 3 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/other"}},
			RepositoryURL:  "s3:bucket/other",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err = db.Create(&policy2).Error
		assert.Error(t, err, "Should fail due to unique constraint on name+tenant_id")
	})

	t.Run("BackupRun CRUD", func(t *testing.T) {
		// Create agent and policy first
		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "backup-test",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		db.Create(&agent)

		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "Backup Policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		db.Create(&policy)

		now := time.Now()
		run := store.BackupRun{
			TenantID:  tenantID,
			AgentID:   agent.ID,
			PolicyID:  policy.ID,
			StartTime: now,
			Status:    "running",
		}

		err := db.Create(&run).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, run.ID)

		// Update with statistics
		endTime := now.Add(10 * time.Minute)
		err = db.Model(&run).Updates(map[string]interface{}{
			"end_time":    endTime,
			"status":      "success",
			"files_new":   1000,
			"data_added":  5000000000,
			"snapshot_id": "abc123",
		}).Error
		require.NoError(t, err)

		var retrieved store.BackupRun
		err = db.First(&retrieved, "id = ?", run.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "success", retrieved.Status)
		assert.NotNil(t, retrieved.EndTime)
		assert.Equal(t, 1000, *retrieved.FilesNew)
	})

	t.Run("BackupRun Upsert", func(t *testing.T) {
		// Create agent and policy first
		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "upsert-test",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		db.Create(&agent)

		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "Upsert Policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		db.Create(&policy)

		// Test upsert - first insert
		taskID := uuid.New()
		now := time.Now()
		duration := 120.5
		snapshotID := "abc123"

		run := store.BackupRun{
			ID:              taskID,
			TenantID:        tenantID,
			AgentID:         agent.ID,
			PolicyID:        policy.ID,
			StartTime:       now,
			EndTime:         &now,
			Status:          "success",
			DurationSeconds: &duration,
			SnapshotID:      &snapshotID,
		}

		err := db.Save(&run).Error
		require.NoError(t, err)

		// Verify created
		var retrieved store.BackupRun
		err = db.First(&retrieved, "id = ?", taskID).Error
		require.NoError(t, err)
		assert.Equal(t, "success", retrieved.Status)
		assert.Equal(t, "abc123", *retrieved.SnapshotID)

		// Test upsert - update existing
		errorMsg := "connection failed"
		run.Status = "failed"
		run.ErrorMessage = &errorMsg
		run.SnapshotID = nil

		err = db.Save(&run).Error
		require.NoError(t, err)

		// Verify updated
		err = db.First(&retrieved, "id = ?", taskID).Error
		require.NoError(t, err)
		assert.Equal(t, "failed", retrieved.Status)
		assert.Equal(t, "connection failed", *retrieved.ErrorMessage)
		assert.Nil(t, retrieved.SnapshotID)
	})

	t.Run("AgentPolicyLink CRUD", func(t *testing.T) {
		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "link-test",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		db.Create(&agent)

		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "Link Policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		db.Create(&policy)

		link := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy.ID,
		}

		err := db.Create(&link).Error
		require.NoError(t, err)

		var retrieved store.AgentPolicyLink
		err = db.First(&retrieved, "agent_id = ? AND policy_id = ?", agent.ID, policy.ID).Error
		require.NoError(t, err)
		assert.Equal(t, agent.ID, retrieved.AgentID)
		assert.Equal(t, policy.ID, retrieved.PolicyID)
	})
}

// TestAgentPolicyLinkDuplicatePrevention tests that duplicate assignments are prevented (TDD - Epic 7)
func TestAgentPolicyLinkDuplicatePrevention(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	err = store.MigrateModels(db)
	require.NoError(t, err)

	tenantID := uuid.New()

	// Create agent and policy
	agent := store.Agent{
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = db.Create(&agent).Error
	require.NoError(t, err)

	policy := store.Policy{
		TenantID:       tenantID,
		Name:           "Test Policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/path",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = db.Create(&policy).Error
	require.NoError(t, err)

	// Create first assignment
	link1 := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err = db.Create(&link1).Error
	require.NoError(t, err, "First assignment should succeed")

	// Attempt duplicate assignment
	link2 := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err = db.Create(&link2).Error
	assert.Error(t, err, "Duplicate assignment should fail due to composite primary key")

	// Verify only one link exists
	var count int64
	db.Model(&store.AgentPolicyLink{}).Where("agent_id = ? AND policy_id = ?", agent.ID, policy.ID).Count(&count)
	assert.Equal(t, int64(1), count, "Only one link should exist")
}

// TestAgentPolicyLinkCascadeDeleteAgent tests cascade delete when agent is deleted (TDD - Epic 7)
func TestAgentPolicyLinkCascadeDeleteAgent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	err = store.MigrateModels(db)
	require.NoError(t, err)

	tenantID := uuid.New()

	// Create agent
	agent := store.Agent{
		TenantID: tenantID,
		Hostname: "cascade-test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = db.Create(&agent).Error
	require.NoError(t, err)

	// Create two policies
	policy1 := store.Policy{
		TenantID:       tenantID,
		Name:           "Policy One",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/path1",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = db.Create(&policy1).Error
	require.NoError(t, err)

	policy2 := store.Policy{
		TenantID:       tenantID,
		Name:           "Policy Two",
		Schedule:       "0 3 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/home"}},
		RepositoryURL:  "s3:bucket/path2",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = db.Create(&policy2).Error
	require.NoError(t, err)

	// Create two assignments
	link1 := store.AgentPolicyLink{AgentID: agent.ID, PolicyID: policy1.ID}
	err = db.Create(&link1).Error
	require.NoError(t, err)

	link2 := store.AgentPolicyLink{AgentID: agent.ID, PolicyID: policy2.ID}
	err = db.Create(&link2).Error
	require.NoError(t, err)

	// Verify assignments exist
	var countBefore int64
	db.Model(&store.AgentPolicyLink{}).Where("agent_id = ?", agent.ID).Count(&countBefore)
	assert.Equal(t, int64(2), countBefore, "Two assignments should exist before deletion")

	// Delete agent
	err = db.Delete(&agent).Error
	require.NoError(t, err)

	// Verify assignments were cascade-deleted
	var countAfter int64
	db.Model(&store.AgentPolicyLink{}).Where("agent_id = ?", agent.ID).Count(&countAfter)
	assert.Equal(t, int64(0), countAfter, "All assignments should be cascade-deleted when agent is deleted")

	// Verify policies still exist
	var policy1Retrieved store.Policy
	err = db.First(&policy1Retrieved, "id = ?", policy1.ID).Error
	assert.NoError(t, err, "Policies should not be deleted when agent is deleted")
}

// TestAgentPolicyLinkCascadeDeletePolicy tests cascade delete when policy is deleted (TDD - Epic 7)
func TestAgentPolicyLinkCascadeDeletePolicy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	err = store.MigrateModels(db)
	require.NoError(t, err)

	tenantID := uuid.New()

	// Create two agents
	agent1 := store.Agent{
		TenantID: tenantID,
		Hostname: "agent-one",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = db.Create(&agent1).Error
	require.NoError(t, err)

	agent2 := store.Agent{
		TenantID: tenantID,
		Hostname: "agent-two",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = db.Create(&agent2).Error
	require.NoError(t, err)

	// Create policy
	policy := store.Policy{
		TenantID:       tenantID,
		Name:           "Cascade Policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/path",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = db.Create(&policy).Error
	require.NoError(t, err)

	// Create two assignments
	link1 := store.AgentPolicyLink{AgentID: agent1.ID, PolicyID: policy.ID}
	err = db.Create(&link1).Error
	require.NoError(t, err)

	link2 := store.AgentPolicyLink{AgentID: agent2.ID, PolicyID: policy.ID}
	err = db.Create(&link2).Error
	require.NoError(t, err)

	// Verify assignments exist
	var countBefore int64
	db.Model(&store.AgentPolicyLink{}).Where("policy_id = ?", policy.ID).Count(&countBefore)
	assert.Equal(t, int64(2), countBefore, "Two assignments should exist before deletion")

	// Delete policy
	err = db.Delete(&policy).Error
	require.NoError(t, err)

	// Verify assignments were cascade-deleted
	var countAfter int64
	db.Model(&store.AgentPolicyLink{}).Where("policy_id = ?", policy.ID).Count(&countAfter)
	assert.Equal(t, int64(0), countAfter, "All assignments should be cascade-deleted when policy is deleted")

	// Verify agents still exist
	var agent1Retrieved store.Agent
	err = db.First(&agent1Retrieved, "id = ?", agent1.ID).Error
	assert.NoError(t, err, "Agents should not be deleted when policy is deleted")
}

// TestAgentPolicyLinkForeignKeyEnforcement tests foreign key constraints (TDD - Epic 7)
func TestAgentPolicyLinkForeignKeyEnforcement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	err = store.MigrateModels(db)
	require.NoError(t, err)

	t.Run("Cannot assign non-existent policy", func(t *testing.T) {
		tenantID := uuid.New()

		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "fk-test-agent",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		err := db.Create(&agent).Error
		require.NoError(t, err)

		// Attempt to link to non-existent policy
		nonExistentPolicyID := uuid.New()
		link := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: nonExistentPolicyID,
		}
		err = db.Create(&link).Error
		assert.Error(t, err, "Should fail due to foreign key constraint on policy_id")
	})

	t.Run("Cannot assign to non-existent agent", func(t *testing.T) {
		tenantID := uuid.New()

		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "FK Test Policy",
			Schedule:       "0 2 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/path",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err := db.Create(&policy).Error
		require.NoError(t, err)

		// Attempt to link to non-existent agent
		nonExistentAgentID := uuid.New()
		link := store.AgentPolicyLink{
			AgentID:  nonExistentAgentID,
			PolicyID: policy.ID,
		}
		err = db.Create(&link).Error
		assert.Error(t, err, "Should fail due to foreign key constraint on agent_id")
	})
}

// TestAgentPolicyLinkMultipleAssignments tests many-to-many relationship (TDD - Epic 7)
func TestAgentPolicyLinkMultipleAssignments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	err = store.MigrateModels(db)
	require.NoError(t, err)

	tenantID := uuid.New()

	t.Run("One agent can have multiple policies", func(t *testing.T) {
		agent := store.Agent{
			TenantID: tenantID,
			Hostname: "multi-policy-agent",
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   "online",
		}
		err := db.Create(&agent).Error
		require.NoError(t, err)

		// Create three policies
		policies := make([]store.Policy, 3)
		for i := 0; i < 3; i++ {
			policies[i] = store.Policy{
				TenantID:       tenantID,
				Name:           "Multi Policy " + string(rune('A'+i)),
				Schedule:       "0 2 * * *",
				IncludePaths:   store.JSONB{"paths": []string{"/data"}},
				RepositoryURL:  "s3:bucket/path" + string(rune('A'+i)),
				RepositoryType: "s3",
				RetentionRules: store.JSONB{"keep_daily": 7},
				Enabled:        true,
			}
			err = db.Create(&policies[i]).Error
			require.NoError(t, err)

			// Assign each policy to agent
			link := store.AgentPolicyLink{
				AgentID:  agent.ID,
				PolicyID: policies[i].ID,
			}
			err = db.Create(&link).Error
			require.NoError(t, err)
		}

		// Verify agent has 3 policies
		var count int64
		db.Model(&store.AgentPolicyLink{}).Where("agent_id = ?", agent.ID).Count(&count)
		assert.Equal(t, int64(3), count, "Agent should have 3 policy assignments")
	})

	t.Run("One policy can be assigned to multiple agents", func(t *testing.T) {
		policy := store.Policy{
			TenantID:       tenantID,
			Name:           "Shared Policy",
			Schedule:       "0 3 * * *",
			IncludePaths:   store.JSONB{"paths": []string{"/data"}},
			RepositoryURL:  "s3:bucket/shared",
			RepositoryType: "s3",
			RetentionRules: store.JSONB{"keep_daily": 7},
			Enabled:        true,
		}
		err := db.Create(&policy).Error
		require.NoError(t, err)

		// Create three agents
		agents := make([]store.Agent, 3)
		for i := 0; i < 3; i++ {
			agents[i] = store.Agent{
				TenantID: tenantID,
				Hostname: "shared-agent-" + string(rune('A'+i)),
				OS:       "linux",
				Arch:     "amd64",
				Version:  "1.0.0",
				Status:   "online",
			}
			err = db.Create(&agents[i]).Error
			require.NoError(t, err)

			// Assign policy to each agent
			link := store.AgentPolicyLink{
				AgentID:  agents[i].ID,
				PolicyID: policy.ID,
			}
			err = db.Create(&link).Error
			require.NoError(t, err)
		}

		// Verify policy is assigned to 3 agents
		var count int64
		db.Model(&store.AgentPolicyLink{}).Where("policy_id = ?", policy.ID).Count(&count)
		assert.Equal(t, int64(3), count, "Policy should be assigned to 3 agents")
	})
}

// TestStoreBackupRunLogs tests storing log entries for backup runs (TDD - Epic 13.4)
func TestStoreBackupRunLogs(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "log-test",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "Log Test Policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create backup run
	backupRun := &store.BackupRun{
		TenantID:  st.GetTenantID(),
		AgentID:   agent.ID,
		PolicyID:  policy.ID,
		StartTime: time.Now(),
		Status:    "running",
	}
	err = st.UpsertBackupRun(ctx, backupRun)
	require.NoError(t, err)

	// Test storing small log (single entry)
	smallLog := "Starting backup...\nProcessing files...\nBackup complete!"
	err = st.StoreBackupRunLogs(ctx, backupRun.ID, smallLog)
	require.NoError(t, err)

	// Retrieve logs
	logs, err := st.GetBackupRunLogs(ctx, backupRun.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 1, "Should have 1 log entry for small log")
	assert.Equal(t, smallLog, logs[0].Message)
	assert.Equal(t, "info", logs[0].Level)
}

// TestStoreBackupRunLogsChunked tests chunking large logs (TDD - Epic 13.4)
func TestStoreBackupRunLogsChunked(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "chunk-test",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "Chunk Test Policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create backup run
	backupRun := &store.BackupRun{
		TenantID:  st.GetTenantID(),
		AgentID:   agent.ID,
		PolicyID:  policy.ID,
		StartTime: time.Now(),
		Status:    "running",
	}
	err = st.UpsertBackupRun(ctx, backupRun)
	require.NoError(t, err)

	// Create a large log (>1MB to trigger chunking)
	largeLog := strings.Repeat("This is a log line that will be repeated many times to create a large log.\n", 20000)
	assert.Greater(t, len(largeLog), 1024*1024, "Log should be >1MB")

	// Store large log
	err = st.StoreBackupRunLogs(ctx, backupRun.ID, largeLog)
	require.NoError(t, err)

	// Retrieve logs
	logs, err := st.GetBackupRunLogs(ctx, backupRun.ID)
	require.NoError(t, err)
	assert.Greater(t, len(logs), 1, "Large log should be chunked into multiple entries")

	// Reconstruct full log
	var reconstructed strings.Builder
	for _, log := range logs {
		reconstructed.WriteString(log.Message)
	}
	assert.Equal(t, largeLog, reconstructed.String(), "Reconstructed log should match original")
}

// TestGetBackupRunLogsOrdering tests log entries are returned in correct order (TDD - Epic 13.4)
func TestGetBackupRunLogsOrdering(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "order-test",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "Order Test Policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create backup run
	backupRun := &store.BackupRun{
		TenantID:  st.GetTenantID(),
		AgentID:   agent.ID,
		PolicyID:  policy.ID,
		StartTime: time.Now(),
		Status:    "running",
	}
	err = st.UpsertBackupRun(ctx, backupRun)
	require.NoError(t, err)

	// Store log in chunks with small delays
	log1 := "First chunk"
	log2 := "Second chunk"
	log3 := "Third chunk"

	time.Sleep(1 * time.Millisecond)
	err = st.StoreBackupRunLogs(ctx, backupRun.ID, log1)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	err = st.StoreBackupRunLogs(ctx, backupRun.ID, log2)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	err = st.StoreBackupRunLogs(ctx, backupRun.ID, log3)
	require.NoError(t, err)

	// Retrieve logs
	logs, err := st.GetBackupRunLogs(ctx, backupRun.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 3, "Should have 3 log entries")

	// Verify ordering (should be chronological)
	assert.Contains(t, logs[0].Message, "First")
	assert.Contains(t, logs[1].Message, "Second")
	assert.Contains(t, logs[2].Message, "Third")
}

// Helper functions for pointers
func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}
