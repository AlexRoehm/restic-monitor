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

// TestGetBackupRunsSuccess tests successful retrieval of backup runs (TDD - Epic 13.6)
func TestGetBackupRunsSuccess(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create backup runs
	now := time.Now()
	duration1 := 100.0
	duration2 := 150.0
	snapshot1 := "snap1"

	run1 := &store.BackupRun{
		TenantID:        st.GetTenantID(),
		AgentID:         agent.ID,
		PolicyID:        policy.ID,
		StartTime:       now.Add(-2 * time.Hour),
		EndTime:         &now,
		Status:          "success",
		DurationSeconds: &duration1,
		SnapshotID:      &snapshot1,
	}
	err = st.UpsertBackupRun(ctx, run1)
	require.NoError(t, err)

	run2 := &store.BackupRun{
		TenantID:        st.GetTenantID(),
		AgentID:         agent.ID,
		PolicyID:        policy.ID,
		StartTime:       now.Add(-1 * time.Hour),
		Status:          "running",
		DurationSeconds: &duration2,
	}
	err = st.UpsertBackupRun(ctx, run2)
	require.NoError(t, err)

	// Request backup runs
	req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/backup-runs", nil)
	w := httptest.NewRecorder()

	api.handleGetBackupRuns(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BackupRunsResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response.Runs, 2, "Should return 2 backup runs")
	assert.Equal(t, "running", response.Runs[0].Status, "Most recent should be first")
	assert.Equal(t, "success", response.Runs[1].Status)
}

// TestGetBackupRunsWithStatusFilter tests filtering by status (TDD - Epic 13.6)
func TestGetBackupRunsWithStatusFilter(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create backup runs with different statuses
	now := time.Now()
	duration := 100.0

	for i, status := range []string{"success", "failed", "success", "running"} {
		run := &store.BackupRun{
			TenantID:        st.GetTenantID(),
			AgentID:         agent.ID,
			PolicyID:        policy.ID,
			StartTime:       now.Add(-time.Duration(i) * time.Hour),
			Status:          status,
			DurationSeconds: &duration,
		}
		err = st.UpsertBackupRun(ctx, run)
		require.NoError(t, err)
	}

	// Request only successful runs
	req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/backup-runs?status=success", nil)
	w := httptest.NewRecorder()

	api.handleGetBackupRuns(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BackupRunsResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response.Runs, 2, "Should return only success runs")
	for _, run := range response.Runs {
		assert.Equal(t, "success", run.Status)
	}
}

// TestGetBackupRunsWithPagination tests limit and offset (TDD - Epic 13.6)
func TestGetBackupRunsWithPagination(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
		Schedule:       "0 0 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3://bucket/repo",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keepLast": 7},
		Enabled:        true,
	}
	err = st.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create 10 backup runs
	now := time.Now()
	duration := 100.0

	for i := 0; i < 10; i++ {
		run := &store.BackupRun{
			TenantID:        st.GetTenantID(),
			AgentID:         agent.ID,
			PolicyID:        policy.ID,
			StartTime:       now.Add(-time.Duration(i) * time.Hour),
			Status:          "success",
			DurationSeconds: &duration,
		}
		err = st.UpsertBackupRun(ctx, run)
		require.NoError(t, err)
	}

	// Request with limit
	req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/backup-runs?limit=5", nil)
	w := httptest.NewRecorder()

	api.handleGetBackupRuns(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BackupRunsResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response.Runs, 5, "Should return only 5 runs")
	assert.Equal(t, 10, response.Total, "Total should be 10")
}

// TestGetBackupRunsNonexistentAgent tests 404 for unknown agent (TDD - Epic 13.6)
func TestGetBackupRunsNonexistentAgent(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")

	fakeAgentID := uuid.New()
	req := httptest.NewRequest("GET", "/agents/"+fakeAgentID.String()+"/backup-runs", nil)
	w := httptest.NewRecorder()

	api.handleGetBackupRuns(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetBackupRunWithLogs tests retrieving a single run with logs (TDD - Epic 13.6)
func TestGetBackupRunWithLogs(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)
	api := New(config.Config{}, st, nil, "")

	ctx := context.Background()

	// Create agent and policy
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.CreateAgent(ctx, agent)
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       st.GetTenantID(),
		Name:           "test-policy",
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
	now := time.Now()
	duration := 100.0
	snapshot := "snap123"

	run := &store.BackupRun{
		TenantID:        st.GetTenantID(),
		AgentID:         agent.ID,
		PolicyID:        policy.ID,
		StartTime:       now.Add(-1 * time.Hour),
		EndTime:         &now,
		Status:          "success",
		DurationSeconds: &duration,
		SnapshotID:      &snapshot,
	}
	err = st.UpsertBackupRun(ctx, run)
	require.NoError(t, err)

	// Store logs
	logContent := "Backup started\nProcessing files\nBackup completed"
	err = st.StoreBackupRunLogs(ctx, run.ID, logContent)
	require.NoError(t, err)

	// Request single backup run with logs
	req := httptest.NewRequest("GET", "/agents/"+agent.ID.String()+"/backup-runs/"+run.ID.String(), nil)
	w := httptest.NewRecorder()

	api.handleGetBackupRun(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response BackupRunDetailResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, run.ID, response.ID)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, logContent, response.Log, "Should include full log content")
}
