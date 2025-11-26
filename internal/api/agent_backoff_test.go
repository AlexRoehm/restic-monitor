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

// setupTestAPIForBackoff creates a test API instance
func setupTestAPIForBackoff(t *testing.T) (*API, *store.Store) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	
	cfg := config.Config{}
	api := New(cfg, st, nil, "")
	
	return api, st
}

// TestGetAgentBackoffStatus tests retrieving agent backoff state
func TestGetAgentBackoffStatus(t *testing.T) {
	api, st := setupTestAPIForBackoff(t)
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

	// Create policy
	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create tasks in backoff
	futureRetry1 := time.Now().Add(5 * time.Minute)
	futureRetry2 := time.Now().Add(10 * time.Minute)
	retryCount1 := 1
	retryCount2 := 2
	maxRetries := 3
	errorCat := "network"

	task1 := &store.Task{
		TenantID:          st.GetTenantID(),
		AgentID:           agent.ID,
		PolicyID:          policy.ID,
		TaskType:          "backup",
		Status:            "pending",
		Repository:        "s3://bucket/repo",
		RetryCount:        &retryCount1,
		MaxRetries:        &maxRetries,
		NextRetryAt:       &futureRetry1,
		LastErrorCategory: &errorCat,
	}
	err = st.CreateTask(ctx, task1)
	require.NoError(t, err)

	task2 := &store.Task{
		TenantID:    st.GetTenantID(),
		AgentID:     agent.ID,
		PolicyID:    policy.ID,
		TaskType:    "check",
		Status:      "pending",
		Repository:  "s3://bucket/repo",
		RetryCount:  &retryCount2,
		MaxRetries:  &maxRetries,
		NextRetryAt: &futureRetry2,
	}
	err = st.CreateTask(ctx, task2)
	require.NoError(t, err)

	// Create task not in backoff (past retry time)
	pastRetry := time.Now().Add(-1 * time.Minute)
	task3 := &store.Task{
		TenantID:    st.GetTenantID(),
		AgentID:     agent.ID,
		PolicyID:    policy.ID,
		TaskType:    "backup",
		Status:      "pending",
		Repository:  "s3://bucket/repo",
		NextRetryAt: &pastRetry,
	}
	err = st.CreateTask(ctx, task3)
	require.NoError(t, err)

	// Get backoff status
	req := httptest.NewRequest(http.MethodGet, "/agents/"+agent.ID.String()+"/backoff-status", nil)
	w := httptest.NewRecorder()

	api.handleGetAgentBackoff(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response AgentBackoffResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, agent.ID, response.AgentID)
	assert.Equal(t, "test-agent", response.Hostname)
	assert.Equal(t, 2, response.TasksInBackoff, "Should have 2 tasks in backoff")
	assert.NotNil(t, response.EarliestRetryAt)
	assert.Len(t, response.BackoffTasks, 2)

	// Verify earliest retry is task1's time
	assert.WithinDuration(t, futureRetry1, *response.EarliestRetryAt, time.Second)

	// Verify task details
	found := 0
	for _, task := range response.BackoffTasks {
		if task.TaskID == task1.ID {
			assert.Equal(t, "backup", task.TaskType)
			assert.Equal(t, 1, task.RetryCount)
			assert.Equal(t, 3, task.MaxRetries)
			assert.Equal(t, "network", task.ErrorCategory)
			found++
		}
		if task.TaskID == task2.ID {
			assert.Equal(t, "check", task.TaskType)
			assert.Equal(t, 2, task.RetryCount)
			found++
		}
	}
	assert.Equal(t, 2, found, "Should find both backoff tasks")
}

// TestUpdateAgentBackoffState tests the backoff state calculation
func TestUpdateAgentBackoffState(t *testing.T) {
	api, st := setupTestAPIForBackoff(t)
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

	// Create policy
	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create tasks in backoff
	futureRetry1 := time.Now().Add(5 * time.Minute)
	futureRetry2 := time.Now().Add(10 * time.Minute)
	retryCount := 1
	maxRetries := 3

	for i := 0; i < 3; i++ {
		nextRetry := futureRetry1
		if i == 2 {
			nextRetry = futureRetry2
		}
		task := &store.Task{
			TenantID:    st.GetTenantID(),
			AgentID:     agent.ID,
			PolicyID:    policy.ID,
			TaskType:    "backup",
			Status:      "pending",
			Repository:  "s3://bucket/repo",
			RetryCount:  &retryCount,
			MaxRetries:  &maxRetries,
			NextRetryAt: &nextRetry,
		}
		err = st.CreateTask(ctx, task)
		require.NoError(t, err)
	}

	// Update backoff state
	err = api.UpdateAgentBackoffState(agent.ID)
	require.NoError(t, err)

	// Verify agent was updated
	var updatedAgent store.Agent
	err = st.GetDB().First(&updatedAgent, agent.ID).Error
	require.NoError(t, err)

	assert.NotNil(t, updatedAgent.TasksInBackoff)
	assert.Equal(t, 3, *updatedAgent.TasksInBackoff)
	assert.NotNil(t, updatedAgent.EarliestRetryAt)
	assert.WithinDuration(t, futureRetry1, *updatedAgent.EarliestRetryAt, time.Second)
}

// TestGetAgentBackoffStatusNoTasks tests backoff status with no tasks
func TestGetAgentBackoffStatusNoTasks(t *testing.T) {
	api, st := setupTestAPIForBackoff(t)
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

	// Get backoff status
	req := httptest.NewRequest(http.MethodGet, "/agents/"+agent.ID.String()+"/backoff-status", nil)
	w := httptest.NewRecorder()

	api.handleGetAgentBackoff(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response AgentBackoffResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.TasksInBackoff)
	assert.Nil(t, response.EarliestRetryAt)
	assert.Len(t, response.BackoffTasks, 0)
}

// TestGetAgentBackoffStatusAgentNotFound tests 404 response
func TestGetAgentBackoffStatusAgentNotFound(t *testing.T) {
	api, _ := setupTestAPIForBackoff(t)

	agentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String()+"/backoff-status", nil)
	w := httptest.NewRecorder()

	api.handleGetAgentBackoff(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
