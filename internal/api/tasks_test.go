package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAPIForTasks creates a test API instance with in-memory database
func setupTestAPIForTasks(t *testing.T) (*API, *store.Store) {
	// Create store with in-memory SQLite (migrations run automatically)
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken: "test-token",
	}

	api := New(cfg, st, nil, "")

	return api, st
}

// TestGetAgentTasks_NoPendingTasks tests retrieving tasks when none exist (TDD - Epic 10.2)
func TestGetAgentTasks_NoPendingTasks(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	agentID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []store.Task `json:"tasks"`
	}

	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 0, len(response.Tasks), "Expected 0 tasks when none exist")
}

// TestGetAgentTasks_PendingTasks tests retrieving pending tasks (TDD - Epic 10.2)
func TestGetAgentTasks_PendingTasks(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create pending tasks
	db := st.GetDB()
	tasks := []*store.Task{
		{
			TenantID:   tenantID,
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "backup",
			Status:     "pending",
			Repository: "s3:bucket/repo1",
		},
		{
			TenantID:   tenantID,
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "check",
			Status:     "pending",
			Repository: "s3:bucket/repo2",
		},
	}

	for _, task := range tasks {
		require.NoError(t, db.Create(task).Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []store.Task `json:"tasks"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 2, len(response.Tasks), "Expected 2 tasks")

	// Verify tasks were assigned
	for _, task := range response.Tasks {
		assert.Equal(t, "assigned", task.Status, "Expected status 'assigned'")
		assert.NotNil(t, task.AssignedAt, "AssignedAt should be set")
	}
}

// TestGetAgentTasks_OnlyPendingTasksReturned tests that only pending tasks are returned (TDD - Epic 10.2)
func TestGetAgentTasks_OnlyPendingTasksReturned(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create tasks with different statuses
	db := st.GetDB()
	now := time.Now()
	tasks := []*store.Task{
		{
			TenantID:   tenantID,
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "backup",
			Status:     "pending",
			Repository: "s3:bucket/repo1",
		},
		{
			TenantID:   tenantID,
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "check",
			Status:     "assigned",
			Repository: "s3:bucket/repo2",
			AssignedAt: &now,
		},
		{
			TenantID:    tenantID,
			AgentID:     agentID,
			PolicyID:    uuid.New(),
			TaskType:    "prune",
			Status:      "completed",
			Repository:  "s3:bucket/repo3",
			CompletedAt: &now,
		},
	}

	for _, task := range tasks {
		require.NoError(t, db.Create(task).Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []store.Task `json:"tasks"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Only the pending task should be returned and assigned
	assert.Equal(t, 1, len(response.Tasks), "Expected 1 pending task")
	assert.Equal(t, "assigned", response.Tasks[0].Status, "Expected status 'assigned'")
	assert.Equal(t, "backup", response.Tasks[0].TaskType)
}

// TestGetAgentTasks_InvalidAgentID tests invalid agent ID format (TDD - Epic 10.2)
func TestGetAgentTasks_InvalidAgentID(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	req := httptest.NewRequest(http.MethodGet, "/agents/invalid-uuid/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 for invalid UUID")
}

// TestGetAgentTasks_LimitParameter tests limit query parameter (TDD - Epic 10.2)
func TestGetAgentTasks_LimitParameter(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create 5 pending tasks
	db := st.GetDB()
	for i := 0; i < 5; i++ {
		task := &store.Task{
			TenantID:   tenantID,
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "backup",
			Status:     "pending",
			Repository: "s3:bucket/repo",
		}
		require.NoError(t, db.Create(task).Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks?limit=2", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []store.Task `json:"tasks"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 2, len(response.Tasks), "Expected 2 tasks due to limit parameter")
}

// TestGetAgentTasks_OrderByScheduledTime tests tasks ordered by scheduled_for (TDD - Epic 10.2)
func TestGetAgentTasks_OrderByScheduledTime(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create tasks with different scheduled times
	db := st.GetDB()
	now := time.Now()
	laterTime := now.Add(1 * time.Hour)
	earlierTime := now.Add(-1 * time.Hour)

	tasks := []*store.Task{
		{
			TenantID:     tenantID,
			AgentID:      agentID,
			PolicyID:     uuid.New(),
			TaskType:     "backup",
			Status:       "pending",
			Repository:   "s3:bucket/repo1",
			ScheduledFor: &laterTime,
		},
		{
			TenantID:     tenantID,
			AgentID:      agentID,
			PolicyID:     uuid.New(),
			TaskType:     "check",
			Status:       "pending",
			Repository:   "s3:bucket/repo2",
			ScheduledFor: &earlierTime,
		},
		{
			TenantID:     tenantID,
			AgentID:      agentID,
			PolicyID:     uuid.New(),
			TaskType:     "prune",
			Status:       "pending",
			Repository:   "s3:bucket/repo3",
			ScheduledFor: &now,
		},
	}

	for _, task := range tasks {
		require.NoError(t, db.Create(task).Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []store.Task `json:"tasks"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 3, len(response.Tasks), "Expected 3 tasks")

	// Verify tasks are ordered by scheduled_for (earliest first)
	assert.True(t, response.Tasks[0].ScheduledFor.Before(*response.Tasks[1].ScheduledFor),
		"Tasks should be ordered by scheduled_for (earliest first)")
	assert.True(t, response.Tasks[1].ScheduledFor.Before(*response.Tasks[2].ScheduledFor),
		"Tasks should be ordered by scheduled_for (earliest first)")
}

// TestAcknowledgeTask_Success tests successful task acknowledgment (TDD - Epic 10.4)
func TestAcknowledgeTask_Success(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create assigned task
	db := st.GetDB()
	now := time.Now()
	task := &store.Task{
		TenantID:   tenantID,
		AgentID:    agentID,
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "assigned",
		Repository: "s3:bucket/repo",
		AssignedAt: &now,
	}
	require.NoError(t, db.Create(task).Error)

	req := httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/"+task.ID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "acknowledged", response.Status)

	// Verify task status updated
	var updated store.Task
	db.First(&updated, task.ID)

	assert.Equal(t, "in-progress", updated.Status)
	assert.NotNil(t, updated.AcknowledgedAt)
	assert.NotNil(t, updated.StartedAt)
}

// TestAcknowledgeTask_TaskNotFound tests 404 for non-existent task (TDD - Epic 10.4)
func TestAcknowledgeTask_TaskNotFound(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	agentID := uuid.New()
	taskID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/"+taskID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAcknowledgeTask_Idempotent tests duplicate acknowledgment handling (TDD - Epic 10.4)
func TestAcknowledgeTask_Idempotent(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create in-progress task (already acknowledged)
	db := st.GetDB()
	now := time.Now()
	task := &store.Task{
		TenantID:       tenantID,
		AgentID:        agentID,
		PolicyID:       uuid.New(),
		TaskType:       "backup",
		Status:         "in-progress",
		Repository:     "s3:bucket/repo",
		AssignedAt:     &now,
		AcknowledgedAt: &now,
		StartedAt:      &now,
	}
	require.NoError(t, db.Create(task).Error)

	// Acknowledge again
	req := httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/"+task.ID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "acknowledged", response.Status)
}

// TestAcknowledgeTask_InvalidTaskID tests invalid UUID format (TDD - Epic 10.4)
func TestAcknowledgeTask_InvalidTaskID(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	agentID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/invalid-uuid/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAcknowledgeTask_InvalidAgentID tests invalid agent UUID (TDD - Epic 10.4)
func TestAcknowledgeTask_InvalidAgentID(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	taskID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/agents/invalid-uuid/tasks/"+taskID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAcknowledgeTask_WrongAgent tests task assigned to different agent (TDD - Epic 10.4)
func TestAcknowledgeTask_WrongAgent(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	otherAgentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(nil, agent)
	require.NoError(t, err)

	// Create task assigned to agentID
	db := st.GetDB()
	now := time.Now()
	task := &store.Task{
		TenantID:   tenantID,
		AgentID:    agentID,
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "assigned",
		Repository: "s3:bucket/repo",
		AssignedAt: &now,
	}
	require.NoError(t, db.Create(task).Error)

	// Try to acknowledge with different agent
	req := httptest.NewRequest(http.MethodPost, "/agents/"+otherAgentID.String()+"/tasks/"+task.ID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestTaskLogging tests that task operations are logged (TDD - Epic 10.5)
func TestTaskLogging(t *testing.T) {
	api, st := setupTestAPIForTasks(t)

	agentID := uuid.New()
	tenantID := st.GetTenantID()

	// Create agent
	agent := &store.Agent{
		ID:       agentID,
		TenantID: tenantID,
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(context.TODO(), agent)
	require.NoError(t, err)

	// Create pending task
	db := st.GetDB()
	task := &store.Task{
		TenantID:   tenantID,
		AgentID:    agentID,
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "pending",
		Repository: "s3:bucket/repo",
	}
	require.NoError(t, db.Create(task).Error)

	// Test task retrieval logging
	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task was assigned
	var updated store.Task
	db.First(&updated, task.ID)
	assert.Equal(t, "assigned", updated.Status)

	// Test acknowledgment logging
	req = httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/"+task.ID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w = httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task is in-progress
	db.First(&updated, task.ID)
	assert.Equal(t, "in-progress", updated.Status)
}

// TestTaskErrorLogging tests that errors are logged (TDD - Epic 10.5)
func TestTaskErrorLogging(t *testing.T) {
	api, _ := setupTestAPIForTasks(t)

	// Test invalid agent ID logging
	req := httptest.NewRequest(http.MethodGet, "/agents/invalid-uuid/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test invalid task ID in ack logging
	agentID := uuid.New()
	req = httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/invalid-uuid/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w = httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test task not found logging
	taskID := uuid.New()
	req = httptest.NewRequest(http.MethodPost, "/agents/"+agentID.String()+"/tasks/"+taskID.String()+"/ack", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w = httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ========================================
// EPIC 15 Phase 5: Retry Budget Tests
// ========================================

// TestTaskResponseIncludesRetryMetadata tests that task responses include retry fields
func TestTaskResponseIncludesRetryMetadata(t *testing.T) {
	api, st := setupTestAPIForTasks(t)
	ctx := context.Background()

	// Create agent
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	// Create policy with custom max_retries
	maxRetries := 5
	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		MaxRetries:     &maxRetries,
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create task with retry state
	retryCount := 2
	task := &store.Task{
		TenantID:   st.GetTenantID(),
		AgentID:    agent.ID,
		PolicyID:   policy.ID,
		TaskType:   "backup",
		Status:     "pending",
		Repository: "s3://bucket/repo",
		RetryCount: &retryCount,
		MaxRetries: &maxRetries,
	}
	err = st.CreateTask(ctx, task)
	require.NoError(t, err)

	// Get tasks
	req := httptest.NewRequest(http.MethodGet, "/agents/"+agent.ID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Tasks []TaskResponse `json:"tasks"`
	}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response.Tasks, 1)

	// Verify retry metadata
	taskResp := response.Tasks[0]
	assert.NotNil(t, taskResp.RetryCount)
	assert.Equal(t, 2, *taskResp.RetryCount)
	assert.NotNil(t, taskResp.MaxRetries)
	assert.Equal(t, 5, *taskResp.MaxRetries)
	assert.NotNil(t, taskResp.RetriesRemaining)
	assert.Equal(t, 3, *taskResp.RetriesRemaining, "Should have 3 retries remaining (5 - 2)")
}

// TestPolicyMaxRetriesInheritedByTasks tests that tasks inherit max_retries from policy
func TestPolicyMaxRetriesInheritedByTasks(t *testing.T) {
	_, st := setupTestAPIForTasks(t)
	ctx := context.Background()

	// Create agent
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	// Create policy with custom max_retries
	customMaxRetries := 10
	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "high-retry-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		MaxRetries:     &customMaxRetries,
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create agent-policy link
	link := &store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err = st.GetDB().Create(link).Error
	require.NoError(t, err)

	// Create task directly via store to simulate scheduler behavior
	task := &store.Task{
		TenantID:   st.GetTenantID(),
		AgentID:    agent.ID,
		PolicyID:   policy.ID,
		TaskType:   "backup",
		Status:     "pending",
		Repository: policy.RepositoryURL,
		MaxRetries: policy.MaxRetries, // Scheduler sets this from policy
	}
	err = st.CreateTask(ctx, task)
	require.NoError(t, err)

	// Verify task has inherited max_retries
	var loadedTask store.Task
	err = st.GetDB().First(&loadedTask, task.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, loadedTask.MaxRetries)
	assert.Equal(t, 10, *loadedTask.MaxRetries, "Task should inherit policy's max_retries")
}

// TestRetriesRemainingCalculation tests retries_remaining calculation edge cases
func TestRetriesRemainingCalculation(t *testing.T) {
	tests := []struct {
		name              string
		retryCount        *int
		maxRetries        *int
		expectedRemaining *int
	}{
		{
			name:              "No retries used",
			retryCount:        intPtr(0),
			maxRetries:        intPtr(3),
			expectedRemaining: intPtr(3),
		},
		{
			name:              "Some retries used",
			retryCount:        intPtr(2),
			maxRetries:        intPtr(5),
			expectedRemaining: intPtr(3),
		},
		{
			name:              "All retries used",
			retryCount:        intPtr(3),
			maxRetries:        intPtr(3),
			expectedRemaining: intPtr(0),
		},
		{
			name:              "Over limit (shouldn't happen but handle gracefully)",
			retryCount:        intPtr(5),
			maxRetries:        intPtr(3),
			expectedRemaining: intPtr(0),
		},
		{
			name:              "Nil fields",
			retryCount:        nil,
			maxRetries:        nil,
			expectedRemaining: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := store.Task{
				RetryCount: tt.retryCount,
				MaxRetries: tt.maxRetries,
			}
			resp := buildTaskResponse(task)

			if tt.expectedRemaining == nil {
				assert.Nil(t, resp.RetriesRemaining)
			} else {
				require.NotNil(t, resp.RetriesRemaining)
				assert.Equal(t, *tt.expectedRemaining, *resp.RetriesRemaining)
			}
		})
	}
}
