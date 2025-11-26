package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// EPIC 13.2: Log Ingestion Endpoint Tests
// ========================================

// TestTaskResultIngestionSuccess tests successful result ingestion (TDD - Epic 13.2)
func TestTaskResultIngestionSuccess(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	// Create test agent and policy
	tenantID := uuid.New()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(nil, &policy)
	require.NoError(t, err)

	// Prepare task result payload
	taskID := uuid.New()
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 125.5,
		"log":             "Files: 100 new, 50 changed, 200 unmodified\nSnapshot abc123 saved",
		"snapshotId":      "abc123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

// TestTaskResultIngestionInvalidJSON tests rejection of malformed JSON (TDD - Epic 13.2)
func TestTaskResultIngestionInvalidJSON(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	agentID := uuid.New()
	req := httptest.NewRequest("POST", "/agents/"+agentID.String()+"/task-results", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTaskResultIngestionNonexistentAgent tests 404 for unknown agent (TDD - Epic 13.2)
func TestTaskResultIngestionNonexistentAgent(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	agentID := uuid.New()
	payload := map[string]interface{}{
		"taskId":          uuid.New().String(),
		"policyId":        uuid.New().String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 60.0,
		"log":             "test log",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agentID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestTaskResultIngestionMissingFields tests validation of required fields (TDD - Epic 13.2)
func TestTaskResultIngestionMissingFields(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	// Create test agent
	tenantID := uuid.New()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, &agent)
	require.NoError(t, err)

	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name: "missing taskId",
			payload: map[string]interface{}{
				"policyId":        uuid.New().String(),
				"taskType":        "backup",
				"status":          "success",
				"durationSeconds": 60.0,
			},
		},
		{
			name: "missing taskType",
			payload: map[string]interface{}{
				"taskId":          uuid.New().String(),
				"policyId":        uuid.New().String(),
				"status":          "success",
				"durationSeconds": 60.0,
			},
		},
		{
			name: "missing status",
			payload: map[string]interface{}{
				"taskId":          uuid.New().String(),
				"policyId":        uuid.New().String(),
				"taskType":        "backup",
				"durationSeconds": 60.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			api.handleTaskResults(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestTaskResultIngestionIdempotent tests duplicate submission handling (TDD - Epic 13.2)
func TestTaskResultIngestionIdempotent(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	// Create test agent and policy
	tenantID := uuid.New()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(nil, &policy)
	require.NoError(t, err)

	// Submit same task result twice
	taskID := uuid.New()
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 100.0,
		"log":             "First submission",
		"snapshotId":      "snap001",
	}

	body, _ := json.Marshal(payload)

	// First submission
	req1 := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	api.handleTaskResults(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second submission (duplicate)
	req2 := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	api.handleTaskResults(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify only one record exists (or updated record)
	// This would require a GetBackupRun method in store
}

// TestTaskResultIngestionLargeLog tests handling of large log payloads (TDD - Epic 13.2)
func TestTaskResultIngestionLargeLog(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	// Create test agent and policy
	tenantID := uuid.New()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(nil, &policy)
	require.NoError(t, err)

	// Create a large log (1MB)
	largeLog := bytes.Repeat([]byte("Log line with content\n"), 50000)

	payload := map[string]interface{}{
		"taskId":          uuid.New().String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 300.0,
		"log":             string(largeLog),
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestTaskResultWithLogStorage tests that logs are stored (TDD - Epic 13.4)
func TestTaskResultWithLogStorage(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	_ = st // Used in test

	// Create test agent and policy
	tenantID := uuid.New()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(nil, &policy)
	require.NoError(t, err)

	// Submit task result with log
	taskID := uuid.New()
	logContent := "Starting backup...\nProcessing files...\nBackup complete!"

	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 100.0,
		"log":             logContent,
		"snapshotId":      "snap123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify backup run was created
	backupRun, err := st.GetBackupRun(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "success", backupRun.Status)

	// Verify logs were stored
	logs, err := st.GetBackupRunLogs(context.Background(), taskID)
	require.NoError(t, err)
	assert.Len(t, logs, 1, "Should have stored 1 log entry")
	assert.Equal(t, logContent, logs[0].Message)
	assert.Equal(t, "info", logs[0].Level)
}

// setupTestAPIForTaskResults creates a test API instance with in-memory store
func setupTestAPIForTaskResults(t *testing.T) (*API, *store.Store) {
	tenantID := uuid.New()
	st, err := store.NewWithTenant(":memory:", tenantID)
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")
	return api, st
}

// ========================================
// EPIC 15 Phase 4: Task Retry Logic Tests
// ========================================

// TestTaskResultRetryOnFailure tests that failed tasks are marked for retry
func TestTaskResultRetryOnFailure(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	ctx := context.Background()

	// Create test agent and policy
	tenantID := st.GetTenantID()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, &policy)
	require.NoError(t, err)

	// Create task
	taskID := uuid.New()
	task := store.Task{
		ID:         taskID,
		TenantID:   tenantID,
		AgentID:    agent.ID,
		PolicyID:   policy.ID,
		TaskType:   "backup",
		Status:     "in-progress",
		Repository: "s3://bucket/repo",
	}
	err = st.CreateTask(ctx, &task)
	require.NoError(t, err)

	// Submit failure result with transient error
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "failed",
		"durationSeconds": 10.5,
		"log":             "Connection timeout",
		"errorMessage":    "network timeout",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was updated with retry state
	var updatedTask store.Task
	err = st.GetDB().Where("id = ?", taskID).First(&updatedTask).Error
	require.NoError(t, err)

	assert.Equal(t, "pending", updatedTask.Status, "Task should be pending for retry")
	assert.NotNil(t, updatedTask.RetryCount)
	assert.Equal(t, 1, *updatedTask.RetryCount, "Retry count should be 1")
	assert.NotNil(t, updatedTask.NextRetryAt, "Next retry time should be set")
	assert.NotNil(t, updatedTask.LastErrorCategory)
	assert.Equal(t, "network", *updatedTask.LastErrorCategory)
}

// TestTaskResultResetRetryOnSuccess tests that successful tasks reset retry state
func TestTaskResultResetRetryOnSuccess(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	ctx := context.Background()

	// Create test agent and policy
	tenantID := st.GetTenantID()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, &policy)
	require.NoError(t, err)

	// Create task with existing retry state
	taskID := uuid.New()
	retryCount := 2
	maxRetries := 3
	errorCategory := "network"
	task := store.Task{
		ID:                taskID,
		TenantID:          tenantID,
		AgentID:           agent.ID,
		PolicyID:          policy.ID,
		TaskType:          "backup",
		Status:            "in-progress",
		Repository:        "s3://bucket/repo",
		RetryCount:        &retryCount,
		MaxRetries:        &maxRetries,
		LastErrorCategory: &errorCategory,
	}
	err = st.CreateTask(ctx, &task)
	require.NoError(t, err)

	// Submit success result
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "success",
		"durationSeconds": 45.2,
		"log":             "Backup completed successfully",
		"snapshotId":      "snap123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify retry state was reset
	var updatedTask store.Task
	err = st.GetDB().Where("id = ?", taskID).First(&updatedTask).Error
	require.NoError(t, err)

	assert.Equal(t, "completed", updatedTask.Status)
	assert.NotNil(t, updatedTask.RetryCount)
	assert.Equal(t, 0, *updatedTask.RetryCount, "Retry count should be reset to 0")
	assert.Nil(t, updatedTask.NextRetryAt, "Next retry time should be nil")
	assert.Nil(t, updatedTask.LastErrorCategory, "Error category should be cleared")
	assert.NotNil(t, updatedTask.CompletedAt)
}

// TestTaskResultMaxRetriesExceeded tests that tasks exceeding max retries are marked as failed
func TestTaskResultMaxRetriesExceeded(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	ctx := context.Background()

	// Create test agent and policy
	tenantID := st.GetTenantID()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, &policy)
	require.NoError(t, err)

	// Create task at max retries
	taskID := uuid.New()
	retryCount := 3
	maxRetries := 3
	task := store.Task{
		ID:         taskID,
		TenantID:   tenantID,
		AgentID:    agent.ID,
		PolicyID:   policy.ID,
		TaskType:   "backup",
		Status:     "in-progress",
		Repository: "s3://bucket/repo",
		RetryCount: &retryCount,
		MaxRetries: &maxRetries,
	}
	err = st.CreateTask(ctx, &task)
	require.NoError(t, err)

	// Submit failure result
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "failed",
		"durationSeconds": 5.0,
		"errorMessage":    "timeout",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task is marked as permanently failed
	var updatedTask store.Task
	err = st.GetDB().Where("id = ?", taskID).First(&updatedTask).Error
	require.NoError(t, err)

	assert.Equal(t, "failed", updatedTask.Status, "Task should be permanently failed")
	assert.NotNil(t, updatedTask.CompletedAt, "Completed time should be set")
}

// TestTaskResultPermanentError tests that permanent errors don't trigger retry
func TestTaskResultPermanentError(t *testing.T) {
	api, st := setupTestAPIForTaskResults(t)
	ctx := context.Background()

	// Create test agent and policy
	tenantID := st.GetTenantID()
	agent := store.Agent{
		ID:       uuid.New(),
		TenantID: tenantID,
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, &agent)
	require.NoError(t, err)

	policy := store.Policy{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, &policy)
	require.NoError(t, err)

	// Create task
	taskID := uuid.New()
	task := store.Task{
		ID:         taskID,
		TenantID:   tenantID,
		AgentID:    agent.ID,
		PolicyID:   policy.ID,
		TaskType:   "backup",
		Status:     "in-progress",
		Repository: "s3://bucket/repo",
	}
	err = st.CreateTask(ctx, &task)
	require.NoError(t, err)

	// Submit failure result with permanent error
	payload := map[string]interface{}{
		"taskId":          taskID.String(),
		"policyId":        policy.ID.String(),
		"taskType":        "backup",
		"status":          "failed",
		"durationSeconds": 2.0,
		"errorMessage":    "permission denied: access to repository forbidden",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/"+agent.ID.String()+"/task-results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleTaskResults(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task is marked as permanently failed (no retry)
	var updatedTask store.Task
	err = st.GetDB().Where("id = ?", taskID).First(&updatedTask).Error
	require.NoError(t, err)

	assert.Equal(t, "failed", updatedTask.Status, "Task should be permanently failed")
	assert.Nil(t, updatedTask.NextRetryAt, "Should not schedule retry for permanent error")
	assert.NotNil(t, updatedTask.CompletedAt)
}

// TestCategorizeError tests error categorization
func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		error    string
		expected string
	}{
		{"Network timeout", "connection timeout", "network"},
		{"Network refused", "connection refused", "network"},
		{"Locked repo", "repository is locked", "transient"},
		{"Permission denied", "permission denied", "permission"},
		{"Access denied", "access denied", "permission"},
		{"Not found", "repository not found", "repository"},
		{"Invalid repo", "invalid repository", "repository"},
		{"Unknown error", "something went wrong", "unknown"},
		{"Empty error", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.error)
			assert.Equal(t, tt.expected, result)
		})
	}
}
