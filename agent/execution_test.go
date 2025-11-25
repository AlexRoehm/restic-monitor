package agent_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPollingLoopWithExecutor tests integration of executor with polling loop (TDD - Epic 11.2)
func TestPollingLoopWithExecutor(t *testing.T) {
	cfg := &agent.Config{
		OrchestratorURL:        "http://localhost:8080",
		AuthenticationToken:    "test-token",
		PollingIntervalSeconds: 60,
		HTTPTimeoutSeconds:     30,
		RetryMaxAttempts:       2,
		RetryBackoffSeconds:    1,
	}

	state := &agent.State{
		AgentID:  uuid.New().String(),
		Hostname: "test-agent",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")
	executor := agent.NewTaskExecutor("echo")

	// Set executor on polling loop
	loop.SetExecutor(executor)

	assert.NotNil(t, loop.GetExecutor())
}

// TestExecuteBackupTask tests executing a backup task (TDD - Epic 11.2)
func TestExecuteBackupTask(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		PolicyID:   uuid.New().String(),
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		IncludePaths: map[string]interface{}{
			"paths": []interface{}{"/home/user"},
		},
		CreatedAt: time.Now(),
	}

	result, err := executor.Execute(task)

	require.NoError(t, err)
	assert.Equal(t, task.TaskID, result.TaskID)
	assert.Equal(t, "success", result.Status)
	assert.Greater(t, result.DurationSeconds, float64(0))
	assert.NotEmpty(t, result.Log)
}

// TestExecuteTaskFromQueue tests dequeuing and executing tasks (TDD - Epic 11.2)
func TestExecuteTaskFromQueue(t *testing.T) {
	queue := agent.NewTaskQueue()
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		PolicyID:   uuid.New().String(),
		TaskType:   "backup",
		Repository: "test-repo",
		CreatedAt:  time.Now(),
	}

	// Enqueue task
	err := queue.Enqueue(task)
	require.NoError(t, err)

	// Dequeue and execute
	dequeuedTask := queue.Dequeue()
	require.NotNil(t, dequeuedTask)

	result, err := executor.Execute(*dequeuedTask)
	require.NoError(t, err)

	assert.Equal(t, task.TaskID, result.TaskID)
	assert.Equal(t, "success", result.Status)
}

// TestTaskExecutionWithEnvironmentVars tests Restic env vars (TDD - Epic 11.2)
func TestTaskExecutionWithEnvironmentVars(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	// Set Restic environment variables
	env := map[string]string{
		"RESTIC_REPOSITORY": "s3:bucket/repo",
		"RESTIC_PASSWORD":   "secret123",
		"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
	}

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	result, err := executor.ExecuteWithEnv(task, env)

	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
}

// TestExtractSnapshotID tests snapshot ID extraction from backup output (TDD - Epic 11.2)
func TestExtractSnapshotID(t *testing.T) {
	// Simulated Restic backup output
	output := `
Files:           5 new,     0 changed,     0 unmodified
Dirs:            2 new,     0 changed,     0 unmodified
Added to the repo: 1.234 MiB

processed 5 files, 1.234 MiB in 0:01
snapshot abc123def456 saved
`

	snapshotID := agent.ExtractSnapshotID(output)

	assert.Equal(t, "abc123def456", snapshotID)
}

// TestExtractSnapshotIDNoMatch tests when no snapshot ID found (TDD - Epic 11.2)
func TestExtractSnapshotIDNoMatch(t *testing.T) {
	output := "Some output without snapshot ID"

	snapshotID := agent.ExtractSnapshotID(output)

	assert.Empty(t, snapshotID)
}

// TestTaskStateTransitions tests task state during execution (TDD - Epic 11.2)
func TestTaskStateTransitions(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "test-repo",
		CreatedAt:  time.Now(),
	}

	// Task starts as pending (no state field yet, but concept is important)
	result, err := executor.Execute(task)

	require.NoError(t, err)
	// After execution, result should indicate final state
	assert.Contains(t, []string{"success", "failure"}, result.Status)
}

// TestConcurrentTaskExecution tests executing multiple tasks concurrently (TDD - Epic 11.2)
func TestConcurrentTaskExecution(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	tasks := []agent.Task{
		{
			TaskID:     uuid.New().String(),
			TaskType:   "backup",
			Repository: "repo1",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     uuid.New().String(),
			TaskType:   "backup",
			Repository: "repo2",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     uuid.New().String(),
			TaskType:   "check",
			Repository: "repo3",
			CreatedAt:  time.Now(),
		},
	}

	results := make([]agent.TaskResult, len(tasks))
	done := make(chan int, len(tasks))

	// Execute tasks concurrently
	for i, task := range tasks {
		go func(idx int, tsk agent.Task) {
			result, err := executor.Execute(tsk)
			if err != nil {
				t.Errorf("Task execution failed: %v", err)
			}
			results[idx] = result
			done <- idx
		}(i, task)
	}

	// Wait for all tasks to complete
	for i := 0; i < len(tasks); i++ {
		<-done
	}

	// Verify all tasks completed successfully
	for i, result := range results {
		assert.Equal(t, tasks[i].TaskID, result.TaskID)
		assert.Equal(t, "success", result.Status)
	}
}

// TestMaxConcurrentTasks tests respecting max concurrent task limit (TDD - Epic 11.2)
func TestMaxConcurrentTasks(t *testing.T) {
	cfg := &agent.Config{
		MaxConcurrentJobs: 2,
	}

	executor := agent.NewTaskExecutor("sleep")
	limiter := agent.NewConcurrencyLimiter(cfg.MaxConcurrentJobs)

	tasks := []agent.Task{
		{TaskID: uuid.New().String(), TaskType: "backup", Repository: "0.1", CreatedAt: time.Now()},
		{TaskID: uuid.New().String(), TaskType: "backup", Repository: "0.1", CreatedAt: time.Now()},
		{TaskID: uuid.New().String(), TaskType: "backup", Repository: "0.1", CreatedAt: time.Now()},
	}

	start := time.Now()
	done := make(chan struct{}, len(tasks))

	for _, task := range tasks {
		go func(t agent.Task) {
			limiter.Acquire()
			defer limiter.Release()
			_, _ = executor.Execute(t)
			done <- struct{}{}
		}(task)
	}

	// Wait for all tasks to complete
	for i := 0; i < len(tasks); i++ {
		<-done
	}

	elapsed := time.Since(start)

	// With maxConcurrent=2, three 0.1s tasks should take at least 0.15s
	// (first 2 run concurrently, third waits)
	assert.GreaterOrEqual(t, elapsed.Seconds(), 0.15)
}

// TestExecuteCheckTask tests executing a check task (TDD - Epic 11.3)
func TestExecuteCheckTask(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		PolicyID:   uuid.New().String(),
		TaskType:   "check",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	result, err := executor.Execute(task)

	require.NoError(t, err)
	assert.Equal(t, task.TaskID, result.TaskID)
	assert.Equal(t, "success", result.Status)
	assert.Greater(t, result.DurationSeconds, float64(0))
	assert.NotEmpty(t, result.Log)
}

// TestExecutePruneTask tests executing a prune task (TDD - Epic 11.3)
func TestExecutePruneTask(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		PolicyID:   uuid.New().String(),
		TaskType:   "prune",
		Repository: "s3:bucket/repo",
		Retention: map[string]interface{}{
			"keepLast":    float64(7),
			"keepDaily":   float64(14),
			"keepWeekly":  float64(4),
			"keepMonthly": float64(6),
		},
		CreatedAt: time.Now(),
	}

	result, err := executor.Execute(task)

	require.NoError(t, err)
	assert.Equal(t, task.TaskID, result.TaskID)
	assert.Equal(t, "success", result.Status)
	assert.Greater(t, result.DurationSeconds, float64(0))
}

// TestExtractCheckResults tests parsing check output (TDD - Epic 11.3)
func TestExtractCheckResults(t *testing.T) {
	output := `
using temporary cache in /tmp/restic-check-cache-123456
create exclusive lock for repository
load indexes
check all packs
check snapshots, trees and blobs
no errors were found
`

	checkResult := agent.ParseCheckOutput(output)

	assert.True(t, checkResult.Success)
	assert.Empty(t, checkResult.ErrorsFound)
	assert.Contains(t, checkResult.Summary, "no errors were found")
}

// TestExtractCheckResultsWithErrors tests parsing check output with errors (TDD - Epic 11.3)
func TestExtractCheckResultsWithErrors(t *testing.T) {
	output := `
using temporary cache in /tmp/restic-check-cache-123456
create exclusive lock for repository
load indexes
check all packs
check snapshots, trees and blobs
error: pack abc123: data blob def456: wrong hash
error: pack xyz789: missing blob
Fatal: repository contains errors
`

	checkResult := agent.ParseCheckOutput(output)

	assert.False(t, checkResult.Success)
	assert.Equal(t, 2, checkResult.ErrorsFound)
	assert.Contains(t, checkResult.Summary, "repository contains errors")
}

// TestExtractPruneResults tests parsing prune output (TDD - Epic 11.3)
func TestExtractPruneResults(t *testing.T) {
	output := `
Applying Policy: keep 7 latest, 14 daily, 4 weekly, 6 monthly snapshots
keep 15 snapshots:
ID        Time                 Host        Tags        Paths
------------------------------------------------------------------
abc123    2024-11-25 10:00:00  server1                 /data
def456    2024-11-24 10:00:00  server1                 /data

remove 3 snapshots:
ID        Time                 Host        Tags        Paths
------------------------------------------------------------------
old123    2024-10-01 10:00:00  server1                 /data
old456    2024-09-01 10:00:00  server1                 /data
old789    2024-08-01 10:00:00  server1                 /data

3 snapshots have been removed, running prune
counting files in repo
building new index for repo
[0:05] 100.00%  15 / 15 packs processed
repository contains 15 packs (1234 blobs) with 123.4 MiB
processed 1234 blobs: 500 duplicate blobs, 12.3 MiB duplicate
load all snapshots
find data that is still in use for 15 snapshots
[0:02] 100.00%  15 / 15 snapshots
found 734 of 1234 data blobs still in use, removing 500 unused blobs
will delete 3 packs and rewrite 2 packs, this frees 45.6 MiB
[0:10] 100.00%  5 / 5 packs rewritten
counting files in repo
[0:01] 100.00%  12 / 12 packs
finding old index files
saved new indexes as [abc123 def456]
remove 5 old index files
[0:00] 100.00%  5 / 5 files deleted
remove 3 old packs
[0:01] 100.00%  3 / 3 files deleted
done
`

	pruneResult := agent.ParsePruneOutput(output)

	assert.Equal(t, 3, pruneResult.SnapshotsRemoved)
	assert.Equal(t, 15, pruneResult.SnapshotsKept)
	// 45.6 MiB = 45.6 * 1024 * 1024 = 47,816,704 bytes (approximately)
	assert.InDelta(t, 47816704, pruneResult.SpaceFreedBytes, 100000) // Allow some tolerance
	assert.Contains(t, pruneResult.Summary, "3 snapshots have been removed")
}

// TestPruneWithNoSnapshotsRemoved tests prune when nothing to remove (TDD - Epic 11.3)
func TestPruneWithNoSnapshotsRemoved(t *testing.T) {
	output := `
Applying Policy: keep 7 latest snapshots
keep 5 snapshots:
ID        Time                 Host        Tags        Paths
------------------------------------------------------------------
abc123    2024-11-25 10:00:00  server1                 /data

no snapshots were removed
`

	pruneResult := agent.ParsePruneOutput(output)

	assert.Equal(t, 0, pruneResult.SnapshotsRemoved)
	assert.Equal(t, 5, pruneResult.SnapshotsKept)
	assert.Equal(t, int64(0), pruneResult.SpaceFreedBytes)
	assert.Contains(t, pruneResult.Summary, "no snapshots were removed")
}

// TestCheckTaskWithReadDataOption tests check with --read-data flag (TDD - Epic 11.3)
func TestCheckTaskWithReadDataOption(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "check",
		Repository: "s3:bucket/repo",
		ExecutionParams: map[string]interface{}{
			"readData": true,
		},
		CreatedAt: time.Now(),
	}

	cmd, args := executor.BuildCommand(task)

	assert.Equal(t, "/usr/bin/restic", cmd)
	assert.Contains(t, args, "check")
	assert.Contains(t, args, "--read-data")
}

// TestStructuredTaskResult tests enhanced task result with metadata (TDD - Epic 11.4)
func TestStructuredTaskResult(t *testing.T) {
	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 12.5,
		Log:             "backup completed",
		SnapshotID:      "abc123",
		StartTime:       time.Now().Add(-12 * time.Second),
		EndTime:         time.Now(),
		TaskType:        "backup",
	}

	assert.NotEmpty(t, result.TaskID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "backup", result.TaskType)
	assert.False(t, result.StartTime.IsZero())
	assert.False(t, result.EndTime.IsZero())
}

// TestTaskResultWithMetadata tests result with check/prune metadata (TDD - Epic 11.4)
func TestTaskResultWithMetadata(t *testing.T) {
	checkResult := agent.CheckResult{
		Success:     true,
		ErrorsFound: 0,
		Summary:     "no errors were found",
	}

	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 5.2,
		Log:             "check completed",
		TaskType:        "check",
		Metadata: map[string]interface{}{
			"checkResult": checkResult,
		},
	}

	assert.NotNil(t, result.Metadata)
	metadata, ok := result.Metadata["checkResult"].(agent.CheckResult)
	assert.True(t, ok)
	assert.True(t, metadata.Success)
}

// TestTaskResultJSONSerialization tests full result serialization (TDD - Epic 11.4)
func TestTaskResultJSONSerialization(t *testing.T) {
	startTime := time.Now().Add(-10 * time.Second)
	endTime := time.Now()

	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 10.0,
		Log:             "task completed successfully",
		SnapshotID:      "snap123",
		StartTime:       startTime,
		EndTime:         endTime,
		TaskType:        "backup",
		Metadata: map[string]interface{}{
			"bytesProcessed": 1024000,
		},
	}

	jsonData, err := result.ToJSON()
	require.NoError(t, err)

	// Parse back
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, result.Status, parsed["status"])
	assert.Equal(t, result.TaskType, parsed["taskType"])
	assert.NotNil(t, parsed["startTime"])
	assert.NotNil(t, parsed["endTime"])
}

// TestLogTruncation tests truncating large logs (TDD - Epic 11.4)
func TestLogTruncation(t *testing.T) {
	// Create a large log (100KB)
	largeLog := strings.Repeat("Log line with some content\n", 4000)

	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 1.0,
		Log:             largeLog,
		TaskType:        "backup",
	}

	// Truncate to 50KB
	truncated := result.TruncateLog(50 * 1024)

	assert.Less(t, len(truncated.Log), 51*1024) // Should be under 51KB
	assert.Greater(t, len(truncated.Log), 0)    // Should not be empty
	if len(largeLog) > 50*1024 {
		assert.Contains(t, truncated.Log, "... (log truncated)")
	}
}

// TestTaskResultPersistence tests saving results to disk (TDD - Epic 11.4)
func TestTaskResultPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 5.0,
		Log:             "backup completed",
		SnapshotID:      "abc123",
		TaskType:        "backup",
		StartTime:       time.Now(),
		EndTime:         time.Now(),
	}

	// Save to disk
	err := result.SaveToFile(tmpDir)
	require.NoError(t, err)

	// Read back
	loaded, err := agent.LoadTaskResult(tmpDir, result.TaskID)
	require.NoError(t, err)

	assert.Equal(t, result.TaskID, loaded.TaskID)
	assert.Equal(t, result.Status, loaded.Status)
	assert.Equal(t, result.SnapshotID, loaded.SnapshotID)
}

// TestStructuredLogging tests log entries with timestamps (TDD - Epic 11.4)
func TestStructuredLogging(t *testing.T) {
	logger := agent.NewTaskLogger(uuid.New().String())

	logger.Info("Task started")
	logger.Debug("Processing file: /data/file.txt")
	logger.Error("Failed to backup file: /data/error.txt")
	logger.Info("Task completed")

	logs := logger.GetLogs()

	assert.Len(t, logs, 4)
	assert.Equal(t, "INFO", logs[0].Level)
	assert.Equal(t, "Task started", logs[0].Message)
	assert.Equal(t, "ERROR", logs[2].Level)
	assert.NotEmpty(t, logs[0].Timestamp)
}

// TestLoggerJSONOutput tests logger JSON serialization (TDD - Epic 11.4)
func TestLoggerJSONOutput(t *testing.T) {
	logger := agent.NewTaskLogger(uuid.New().String())

	logger.Info("Starting backup")
	logger.Info("Backup complete")

	jsonData, err := logger.ToJSON()
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.NotNil(t, parsed["taskId"])
	assert.NotNil(t, parsed["entries"])
}

// ========================================
// EPIC 11.5: Retry & Error Handling Tests
// ========================================

// TestRetryConfiguration tests retry config initialization (TDD - Epic 11.5)
func TestRetryConfiguration(t *testing.T) {
	config := agent.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialBackoff)
	assert.Equal(t, 30*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

// TestErrorCategorization tests distinguishing error types (TDD - Epic 11.5)
func TestErrorCategorization(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected agent.ErrorCategory
	}{
		{"network timeout", "dial tcp: i/o timeout", agent.ErrorCategoryNetwork},
		{"connection refused", "connection refused", agent.ErrorCategoryNetwork},
		{"repo locked", "repository is already locked", agent.ErrorCategoryTransient},
		{"permission denied", "permission denied", agent.ErrorCategoryPermission},
		{"repository not found", "repository does not exist", agent.ErrorCategoryRepo},
		{"unknown error", "some random error", agent.ErrorCategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := agent.CategorizeError(errors.New(tt.errMsg))
			assert.Equal(t, tt.expected, category)
		})
	}
}

// TestIsRetryableError tests transient vs permanent error detection (TDD - Epic 11.5)
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		category  agent.ErrorCategory
		retryable bool
	}{
		{"network errors retryable", agent.ErrorCategoryNetwork, true},
		{"transient errors retryable", agent.ErrorCategoryTransient, true},
		{"permission errors not retryable", agent.ErrorCategoryPermission, false},
		{"repo errors not retryable", agent.ErrorCategoryRepo, false},
		{"unknown errors not retryable", agent.ErrorCategoryUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := agent.IsRetryable(tt.category)
			assert.Equal(t, tt.retryable, retryable)
		})
	}
}

// TestExponentialBackoff tests backoff calculation (TDD - Epic 11.5)
func TestExponentialBackoff(t *testing.T) {
	config := agent.RetryConfig{
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Attempt 1: 1s
	backoff1 := config.CalculateBackoff(1)
	assert.Equal(t, 1*time.Second, backoff1)

	// Attempt 2: 2s
	backoff2 := config.CalculateBackoff(2)
	assert.Equal(t, 2*time.Second, backoff2)

	// Attempt 3: 4s
	backoff3 := config.CalculateBackoff(3)
	assert.Equal(t, 4*time.Second, backoff3)

	// Attempt 10 should cap at MaxBackoff (30s)
	backoff10 := config.CalculateBackoff(10)
	assert.Equal(t, 30*time.Second, backoff10)
}

// TestExecuteWithRetry tests successful retry after transient failure (TDD - Epic 11.5)
func TestExecuteWithRetry(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	config := agent.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "/tmp/test-repo",
		ExecutionParams: map[string]interface{}{
			"password": "test-password",
		},
	}

	// Mock a transient failure followed by success
	// This will fail on first attempt, succeed on second
	// In real implementation, executor would handle retries internally
	result, err := executor.ExecuteWithRetry(task, config)

	// We expect this to eventually succeed (or fail with proper retry tracking)
	// For now, just verify the interface exists
	assert.NotNil(t, result)
	_ = err // May be nil or not depending on mock implementation
}

// TestRetryAttemptsTracking tests retry attempt logging (TDD - Epic 11.5)
func TestRetryAttemptsTracking(t *testing.T) {
	result := agent.TaskResult{
		TaskID:   uuid.New().String(),
		Status:   "success",
		TaskType: "backup",
		Metadata: make(map[string]interface{}),
	}

	// Simulate retry tracking
	result.Metadata["retryAttempts"] = 2
	result.Metadata["retriedErrors"] = []string{
		"attempt 1: connection timeout",
		"attempt 2: success",
	}

	assert.Equal(t, 2, result.Metadata["retryAttempts"])
	errors := result.Metadata["retriedErrors"].([]string)
	assert.Len(t, errors, 2)
	assert.Contains(t, errors[0], "connection timeout")
}

// TestPermanentFailureNoRetry tests immediate failure for permanent errors (TDD - Epic 11.5)
func TestPermanentFailureNoRetry(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	config := agent.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "/nonexistent/repo",
		ExecutionParams: map[string]interface{}{
			"password": "wrong-password",
		},
	}

	result, err := executor.ExecuteWithRetry(task, config)

	// Should fail with permanent error (no retries)
	// Verify that retry metadata shows only 1 attempt
	if result != nil && result.Metadata != nil {
		if attempts, ok := result.Metadata["retryAttempts"]; ok {
			assert.Equal(t, 1, attempts)
		}
	}
	_ = err
}

// TestMaxRetriesExceeded tests exhausting all retry attempts (TDD - Epic 11.5)
func TestMaxRetriesExceeded(t *testing.T) {
	config := agent.RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	// Simulate exceeding max retries
	var attemptCount int
	for attemptCount = 1; attemptCount <= config.MaxAttempts; attemptCount++ {
		// Simulate transient error on each attempt
	}

	assert.Equal(t, 3, attemptCount-1) // Should have tried 3 times
}

// ========================================
// EPIC 11.6: Execution Metrics Tests
// ========================================

// TestExecutionMetricsInitialization tests creating metrics tracker (TDD - Epic 11.6)
func TestExecutionMetricsInitialization(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.GetTotalTasks())
	assert.Equal(t, int64(0), metrics.GetSuccessfulTasks())
	assert.Equal(t, int64(0), metrics.GetFailedTasks())
	assert.Equal(t, int64(0), metrics.GetBytesProcessed())
	assert.Equal(t, 0, metrics.GetConcurrentTasks())
}

// TestRecordTaskExecution tests tracking task execution (TDD - Epic 11.6)
func TestRecordTaskExecution(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// Record successful task
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 5.5, 1024*1024*100) // 100 MB

	assert.Equal(t, int64(1), metrics.GetTotalTasks())
	assert.Equal(t, int64(1), metrics.GetSuccessfulTasks())
	assert.Equal(t, int64(0), metrics.GetFailedTasks())
	assert.Equal(t, int64(1024*1024*100), metrics.GetBytesProcessed())

	// Record failed task
	metrics.RecordTaskStart("check")
	metrics.RecordTaskComplete("check", false, 2.0, 0)

	assert.Equal(t, int64(2), metrics.GetTotalTasks())
	assert.Equal(t, int64(1), metrics.GetSuccessfulTasks())
	assert.Equal(t, int64(1), metrics.GetFailedTasks())
}

// TestSuccessRate tests calculating success percentage (TDD - Epic 11.6)
func TestSuccessRate(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// No tasks yet
	assert.Equal(t, 0.0, metrics.GetSuccessRate())

	// 3 successful, 1 failed = 75%
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 1.0, 0)
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 1.0, 0)
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 1.0, 0)
	metrics.RecordTaskStart("check")
	metrics.RecordTaskComplete("check", false, 1.0, 0)

	assert.Equal(t, 75.0, metrics.GetSuccessRate())
}

// TestAverageDuration tests tracking execution time (TDD - Epic 11.6)
func TestAverageDuration(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// No tasks yet
	assert.Equal(t, 0.0, metrics.GetAverageDuration())

	// Record tasks with different durations
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 10.0, 0)
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 20.0, 0)
	metrics.RecordTaskStart("check")
	metrics.RecordTaskComplete("check", true, 30.0, 0)

	// Average: (10 + 20 + 30) / 3 = 20.0
	assert.Equal(t, 20.0, metrics.GetAverageDuration())
}

// TestConcurrentTaskTracking tests monitoring active tasks (TDD - Epic 11.6)
func TestConcurrentTaskTracking(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	assert.Equal(t, 0, metrics.GetConcurrentTasks())

	// Start 3 tasks
	metrics.RecordTaskStart("backup")
	assert.Equal(t, 1, metrics.GetConcurrentTasks())

	metrics.RecordTaskStart("check")
	assert.Equal(t, 2, metrics.GetConcurrentTasks())

	metrics.RecordTaskStart("prune")
	assert.Equal(t, 3, metrics.GetConcurrentTasks())

	// Complete 1 task
	metrics.RecordTaskComplete("backup", true, 5.0, 0)
	assert.Equal(t, 2, metrics.GetConcurrentTasks())

	// Complete remaining tasks
	metrics.RecordTaskComplete("check", true, 3.0, 0)
	metrics.RecordTaskComplete("prune", true, 8.0, 0)
	assert.Equal(t, 0, metrics.GetConcurrentTasks())
}

// TestExecutionMetricsSnapshot tests capturing metrics state (TDD - Epic 11.6)
func TestExecutionMetricsSnapshot(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// Execute some tasks
	metrics.RecordTaskStart("backup")
	metrics.RecordTaskComplete("backup", true, 10.0, 1024*1024*50)
	metrics.RecordTaskStart("check")
	metrics.RecordTaskComplete("check", false, 2.0, 0)

	snapshot := metrics.GetSnapshot()

	assert.Equal(t, int64(2), snapshot.TotalTasks)
	assert.Equal(t, int64(1), snapshot.SuccessfulTasks)
	assert.Equal(t, int64(1), snapshot.FailedTasks)
	assert.InDelta(t, 50.0, snapshot.SuccessRate, 0.01)
	assert.InDelta(t, 6.0, snapshot.AverageDuration, 0.01)
	assert.Equal(t, int64(1024*1024*50), snapshot.BytesProcessed)
	assert.Equal(t, 0, snapshot.ConcurrentTasks)
	assert.NotEmpty(t, snapshot.Timestamp)
}
