package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/scheduler"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockScheduler implements the Scheduler interface for testing
type mockScheduler struct {
	running bool
	metrics scheduler.MetricsSnapshot
}

func (m *mockScheduler) IsRunning() bool {
	return m.running
}

func (m *mockScheduler) GetMetrics() scheduler.MetricsSnapshot {
	return m.metrics
}

func TestHandleSchedulerStatus(t *testing.T) {
	ctx := context.Background()
	testStore, err := store.New(":memory:")
	require.NoError(t, err)

	// Create test policies
	policy1 := &store.Policy{
		Name:           "backup-policy",
		RepositoryURL:  "s3:bucket/repo1",
		Schedule:       "0 2 * * *",
		Enabled:        true,
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep-last": 5},
	}
	err = testStore.CreatePolicy(ctx, policy1)
	require.NoError(t, err)

	checkSchedule := "0 6 * * *"
	policy2 := &store.Policy{
		Name:           "full-policy",
		RepositoryURL:  "s3:bucket/repo2",
		Schedule:       "every 1h",
		CheckSchedule:  &checkSchedule,
		Enabled:        true,
		IncludePaths:   store.JSONB{"paths": []string{"/app"}},
		RetentionRules: store.JSONB{"keep-last": 10},
	}
	err = testStore.CreatePolicy(ctx, policy2)
	require.NoError(t, err)

	// Create disabled policy
	policy3 := &store.Policy{
		Name:           "disabled-policy",
		RepositoryURL:  "s3:bucket/repo3",
		Schedule:       "every 6h",
		Enabled:        false,
		IncludePaths:   store.JSONB{"paths": []string{"/tmp"}},
		RetentionRules: store.JSONB{"keep-last": 3},
	}
	err = testStore.CreatePolicy(ctx, policy3)
	require.NoError(t, err)

	// Create mock scheduler with metrics
	nextRuns := make(map[uuid.UUID]map[string]time.Time)
	nextRuns[policy1.ID] = map[string]time.Time{
		"backup": time.Now().Add(2 * time.Hour),
	}
	nextRuns[policy2.ID] = map[string]time.Time{
		"backup": time.Now().Add(30 * time.Minute),
		"check":  time.Now().Add(4 * time.Hour),
	}

	mockSched := &mockScheduler{
		running: true,
		metrics: scheduler.MetricsSnapshot{
			TasksGeneratedTotal: 42,
			TasksGeneratedByType: map[string]int64{
				"backup": 30,
				"check":  10,
				"prune":  2,
			},
			LastRunTimestamp:      time.Now().Add(-5 * time.Minute),
			TotalRuns:             100,
			ErrorsTotal:           2,
			LastError:             "test error",
			PoliciesProcessed:     200,
			NextRunsByPolicy:      nextRuns,
			AverageProcessingTime: 150 * time.Millisecond,
		},
	}

	api := NewWithScheduler(config.Config{}, testStore, nil, mockSched, "")
	handler := api.Handler()

	req := httptest.NewRequest(http.MethodGet, "/scheduler/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response SchedulerStatusResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Verify response
	assert.True(t, response.Running)
	assert.Equal(t, int64(100), response.TotalRuns)
	assert.Equal(t, int64(42), response.TasksGenerated)
	assert.Equal(t, int64(2), response.ErrorsTotal)
	assert.Equal(t, "test error", response.LastError)
	assert.Equal(t, 2, response.PoliciesEnabled, "Should have 2 enabled policies")

	// Verify upcoming schedule
	assert.Len(t, response.UpcomingSchedule, 3, "Should have 3 scheduled tasks (2 backup + 1 check)")

	// Verify metrics are included
	assert.Equal(t, int64(30), response.Metrics.TasksGeneratedByType["backup"])
	assert.Equal(t, int64(10), response.Metrics.TasksGeneratedByType["check"])
	assert.Equal(t, int64(2), response.Metrics.TasksGeneratedByType["prune"])
}

func TestHandleSchedulerStatusNoScheduler(t *testing.T) {
	testStore, err := store.New(":memory:")
	require.NoError(t, err)

	// Create API without scheduler
	api := New(config.Config{}, testStore, nil, "")
	handler := api.Handler()

	req := httptest.NewRequest(http.MethodGet, "/scheduler/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should get 404 since route is not registered
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleSchedulerStatusMethodNotAllowed(t *testing.T) {
	testStore, err := store.New(":memory:")
	require.NoError(t, err)

	mockSched := &mockScheduler{
		running: false,
		metrics: scheduler.MetricsSnapshot{},
	}

	api := NewWithScheduler(config.Config{}, testStore, nil, mockSched, "")
	handler := api.Handler()

	req := httptest.NewRequest(http.MethodPost, "/scheduler/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandleSchedulerStatusNotRunning(t *testing.T) {
	testStore, err := store.New(":memory:")
	require.NoError(t, err)

	mockSched := &mockScheduler{
		running: false,
		metrics: scheduler.MetricsSnapshot{
			TasksGeneratedTotal: 0,
			TotalRuns:           0,
		},
	}

	api := NewWithScheduler(config.Config{}, testStore, nil, mockSched, "")
	handler := api.Handler()

	req := httptest.NewRequest(http.MethodGet, "/scheduler/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response SchedulerStatusResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.False(t, response.Running, "Scheduler should not be running")
	assert.Equal(t, int64(0), response.TotalRuns)
	assert.Equal(t, int64(0), response.TasksGenerated)
}

func TestHandleSchedulerStatusEmptySchedule(t *testing.T) {
	testStore, err := store.New(":memory:")
	require.NoError(t, err)

	mockSched := &mockScheduler{
		running: true,
		metrics: scheduler.MetricsSnapshot{
			TasksGeneratedTotal: 5,
			TotalRuns:           10,
			NextRunsByPolicy:    make(map[uuid.UUID]map[string]time.Time),
		},
	}

	api := NewWithScheduler(config.Config{}, testStore, nil, mockSched, "")
	handler := api.Handler()

	req := httptest.NewRequest(http.MethodGet, "/scheduler/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response SchedulerStatusResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.PoliciesEnabled, "Should have no enabled policies")
	assert.Empty(t, response.UpcomingSchedule, "Should have no upcoming schedules")
}
