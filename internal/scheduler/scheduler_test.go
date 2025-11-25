package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerStart(t *testing.T) {
	s := setupTestScheduler(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := s.Start(ctx)
	require.NoError(t, err)

	// Scheduler should be running
	assert.True(t, s.IsRunning())

	s.Stop()
	assert.False(t, s.IsRunning())
}

func TestSchedulerGeneratesTasksForDuePolicies(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create a policy with schedule due now
	policy := &store.Policy{
		Name:           "Test Backup",
		Schedule:       "every 1m",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign policy to an agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler once
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify task was created
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "backup", tasks[0].TaskType)
	assert.Equal(t, policy.ID, tasks[0].PolicyID)
}

func TestSchedulerSkipsDisabledPolicies(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create disabled policy
	policy := &store.Policy{
		Name:           "Disabled Backup",
		Schedule:       "every 1m",
		Enabled:        false,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{

		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// No tasks should be created
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestSchedulerRespectsCronSchedule(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with cron schedule that won't trigger now
	// Schedule for 3 AM (assuming test runs at different time)
	policy := &store.Policy{
		Name:           "Daily Backup",
		Schedule:       "0 3 * * *",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{

		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler (should NOT generate task)
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify task state is created but no task generated yet
	state, err := s.GetPolicyTaskState(ctx, policy.ID, "backup")
	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Nil(t, state.LastRun)
	assert.NotNil(t, state.NextRun)
}

func TestSchedulerTracksLastRun(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy
	policy := &store.Policy{
		Name:           "Test Backup",
		Schedule:       "every 1m",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{

		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler first time
	beforeRun := time.Now()
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Check state was updated
	state, err := s.GetPolicyTaskState(ctx, policy.ID, "backup")
	require.NoError(t, err)
	require.NotNil(t, state.LastRun)
	assert.True(t, state.LastRun.After(beforeRun))
	assert.True(t, state.NextRun.After(*state.LastRun))
}

func TestSchedulerHandlesMultiplePolicies(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create two agents
	agent1 := &store.Agent{

		Hostname: "agent-1",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err := s.store.CreateAgent(ctx, agent1)
	require.NoError(t, err)

	agent2 := &store.Agent{

		Hostname: "agent-2",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent2)
	require.NoError(t, err)

	// Create two policies
	policy1 := &store.Policy{
		Name:           "Policy 1",
		Schedule:       "every 1m",
		Enabled:        true,
		RepositoryURL:  "s3:bucket1",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err = s.store.CreatePolicy(ctx, policy1)
	require.NoError(t, err)

	policy2 := &store.Policy{
		Name:           "Policy 2",
		Schedule:       "every 1m",
		Enabled:        true,
		RepositoryURL:  "s3:bucket2",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err = s.store.CreatePolicy(ctx, policy2)
	require.NoError(t, err)

	// Assign policies to agents
	err = s.store.AssignPolicyToAgent(ctx, policy1.ID, agent1.ID)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy2.ID, agent2.ID)
	require.NoError(t, err)

	// Run scheduler
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Both agents should have tasks
	tasks1, err := s.store.GetPendingTasks(ctx, agent1.ID)
	require.NoError(t, err)
	assert.Len(t, tasks1, 1)

	tasks2, err := s.store.GetPendingTasks(ctx, agent2.ID)
	require.NoError(t, err)
	assert.Len(t, tasks2, 1)
}

func TestSchedulerRecoversFromErrors(t *testing.T) {
	s := setupTestScheduler(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create policy with invalid schedule (should be caught during setup)
	policy := &store.Policy{
		Name:           "Bad Schedule",
		Schedule:       "invalid cron",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Start scheduler
	err = s.Start(ctx)
	require.NoError(t, err)
	defer s.Stop()

	// Give it time to run once
	time.Sleep(150 * time.Millisecond)

	// Scheduler should still be running despite error
	assert.True(t, s.IsRunning())
}

func TestSchedulerHandlesMissedSchedules(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with interval schedule
	policy := &store.Policy{
		Name:           "Hourly Backup",
		Schedule:       "every 1h",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Simulate a past execution by manually setting state
	pastTime := time.Now().Add(-2 * time.Hour) // 2 hours ago
	nextRun := pastTime.Add(1 * time.Hour)     // Should have run 1 hour ago
	err = s.SavePolicyTaskState(ctx, policy.ID, "backup", &pastTime, &nextRun)
	require.NoError(t, err)

	// Run scheduler - should detect missed schedule and generate task immediately
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify task was created
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "backup", tasks[0].TaskType)

	// Verify next run is scheduled in the future
	state, err := s.GetPolicyTaskState(ctx, policy.ID, "backup")
	require.NoError(t, err)
	assert.True(t, state.NextRun.After(time.Now()))
}

func TestSchedulerHandlesMultipleMissedSchedules(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with interval schedule
	policy := &store.Policy{
		Name:           "Frequent Backup",
		Schedule:       "every 15m",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Simulate downtime - last run was 2 hours ago, should have run 8 times since
	pastTime := time.Now().Add(-2 * time.Hour)
	nextRun := pastTime.Add(15 * time.Minute)
	err = s.SavePolicyTaskState(ctx, policy.ID, "backup", &pastTime, &nextRun)
	require.NoError(t, err)

	// Run scheduler - should generate only ONE task for the missed schedules
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify only one task was created (not 8)
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1, "Should create only one task for multiple missed schedules")
}

func TestSchedulerCronMissedSchedule(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with cron schedule (daily at 2 AM)
	policy := &store.Policy{
		Name:           "Daily Backup",
		Schedule:       "0 2 * * *",
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Simulate a missed cron schedule - was supposed to run yesterday
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	yesterdayAt2AM := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 2, 0, 0, 0, yesterday.Location())

	// Set state as if the next run was scheduled for yesterday at 2 AM
	err = s.SavePolicyTaskState(ctx, policy.ID, "backup", nil, &yesterdayAt2AM)
	require.NoError(t, err)

	// Run scheduler - should detect missed cron schedule
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify task was created
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

func TestSchedulerMultipleTaskTypes(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with backup, check, and prune schedules
	checkSchedule := "every 6h"
	pruneSchedule := "0 3 * * 0" // Weekly on Sunday at 3 AM
	policy := &store.Policy{
		Name:           "Full Schedule Policy",
		Schedule:       "every 1h",     // backup
		CheckSchedule:  &checkSchedule, // check
		PruneSchedule:  &pruneSchedule, // prune
		Enabled:        true,
		RepositoryURL:  "s3:backup-bucket",
		RepositoryType: "s3",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep_last": 7},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Assign to agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler - should create backup and check tasks (interval schedules)
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Verify tasks were created
	tasks, err := s.store.GetPendingTasks(ctx, agent.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 2, "Should create backup and check tasks")

	// Check task types
	taskTypes := make(map[string]bool)
	for _, task := range tasks {
		taskTypes[task.TaskType] = true
	}
	assert.True(t, taskTypes["backup"], "Should have backup task")
	assert.True(t, taskTypes["check"], "Should have check task")
	assert.False(t, taskTypes["prune"], "Prune task should not be due yet (cron schedule)")

	// Verify separate states were created
	backupState, err := s.GetPolicyTaskState(ctx, policy.ID, "backup")
	require.NoError(t, err)
	assert.NotNil(t, backupState)

	checkState, err := s.GetPolicyTaskState(ctx, policy.ID, "check")
	require.NoError(t, err)
	assert.NotNil(t, checkState)

	pruneState, err := s.GetPolicyTaskState(ctx, policy.ID, "prune")
	require.NoError(t, err)
	assert.NotNil(t, pruneState)
	assert.Nil(t, pruneState.LastRun, "Prune hasn't run yet")
}

func TestSchedulerMetrics(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	checkSchedule := "every 2h"

	// Create test policy
	policy := &store.Policy{
		Name:           "test-policy",
		RepositoryURL:  "s3:test-bucket/repo",
		Schedule:       "every 1h",
		CheckSchedule:  &checkSchedule,
		Enabled:        true,
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep-last": 5},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create test agent
	agent := &store.Agent{
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "active",
	}
	err = s.store.CreateAgent(ctx, agent)
	require.NoError(t, err)

	err = s.store.AssignPolicyToAgent(ctx, policy.ID, agent.ID)
	require.NoError(t, err)

	// Run scheduler
	err = s.runOnce(ctx)
	require.NoError(t, err)

	// Get metrics snapshot
	metrics := s.GetMetrics()

	// Verify task generation metrics
	assert.Equal(t, int64(2), metrics.TasksGeneratedTotal, "Should have 2 tasks generated (backup + check)")
	assert.Equal(t, int64(1), metrics.TasksGeneratedByType["backup"], "Should have 1 backup task")
	assert.Equal(t, int64(1), metrics.TasksGeneratedByType["check"], "Should have 1 check task")

	// Verify run metrics
	assert.Equal(t, int64(1), metrics.TotalRuns, "Should have 1 scheduler run")
	assert.Equal(t, int64(1), metrics.PoliciesProcessed, "Should have processed 1 policy")
	assert.False(t, metrics.LastRunTimestamp.IsZero(), "Should have last run timestamp")

	// Verify next run tracking
	assert.NotNil(t, metrics.NextRunsByPolicy[policy.ID], "Should have next runs for policy")
	assert.NotNil(t, metrics.NextRunsByPolicy[policy.ID]["backup"], "Should have next run for backup")
	assert.NotNil(t, metrics.NextRunsByPolicy[policy.ID]["check"], "Should have next run for check")

	// Verify no errors
	assert.Equal(t, int64(0), metrics.ErrorsTotal, "Should have no errors")
	assert.Empty(t, metrics.LastError, "Should have no last error")

	// Run again without new tasks (not yet due)
	time.Sleep(10 * time.Millisecond)
	err = s.runOnce(ctx)
	require.NoError(t, err)

	metrics = s.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalRuns, "Should have 2 scheduler runs")
	assert.Equal(t, int64(2), metrics.TasksGeneratedTotal, "Should still have 2 tasks (no new tasks generated)")
}

func TestSchedulerMetricsErrors(t *testing.T) {
	s := setupTestScheduler(t)
	ctx := context.Background()

	// Create policy with invalid schedule
	policy := &store.Policy{
		Name:           "bad-policy",
		RepositoryURL:  "s3:test-bucket/repo",
		Schedule:       "invalid schedule",
		Enabled:        true,
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RetentionRules: store.JSONB{"keep-last": 5},
	}
	err := s.store.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Run scheduler - should encounter error
	_ = s.runOnce(ctx)

	// Get metrics
	metrics := s.GetMetrics()

	// Verify error was recorded
	assert.Greater(t, metrics.ErrorsTotal, int64(0), "Should have recorded errors")
	assert.NotEmpty(t, metrics.LastError, "Should have last error message")
	assert.False(t, metrics.LastErrorTimestamp.IsZero(), "Should have error timestamp")
}

// Helper functions

func setupTestScheduler(t *testing.T) *Scheduler {
	// Create in-memory test database
	testStore, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return NewScheduler(testStore, 100*time.Millisecond)
}
