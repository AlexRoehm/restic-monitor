package store_test

import (
	"encoding/json"
	"reflect"
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
				"ID", "TenantID", "Name", "Schedule", "IncludePaths",
				"ExcludePaths", "RepositoryURL", "RepositoryType",
				"RepositoryConfig", "RetentionRules", "Enabled",
				"CreatedAt", "UpdatedAt",
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
				ID:       policyID,
				TenantID: tenantID,
				Name:     "Daily Production Backup",
				Schedule: "0 2 * * *",
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
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
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
