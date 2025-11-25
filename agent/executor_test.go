package agent_test

import (
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskExecutorConstruction tests TaskExecutor instantiation (TDD - Epic 11.1)
func TestTaskExecutorConstruction(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	assert.NotNil(t, executor)
}

// TestBuildBackupCommand tests CLI command construction for backup (TDD - Epic 11.1)
func TestBuildBackupCommand(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "s3:s3.amazonaws.com/bucket/repo",
		IncludePaths: map[string]interface{}{
			"paths": []interface{}{"/home/user", "/etc"},
		},
		ExcludePaths: map[string]interface{}{
			"paths": []interface{}{"*.tmp", "*.log"},
		},
	}

	cmd, args := executor.BuildCommand(task)

	assert.Equal(t, "/usr/bin/restic", cmd)
	assert.Contains(t, args, "backup")
	assert.Contains(t, args, "/home/user")
	assert.Contains(t, args, "/etc")
	assert.Contains(t, args, "--exclude=*.tmp")
	assert.Contains(t, args, "--exclude=*.log")
}

// TestBuildCheckCommand tests CLI command for check task (TDD - Epic 11.1)
func TestBuildCheckCommand(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "check",
		Repository: "s3:bucket/repo",
	}

	cmd, args := executor.BuildCommand(task)

	assert.Equal(t, "/usr/bin/restic", cmd)
	assert.Contains(t, args, "check")
	assert.Contains(t, args, "-r")
	assert.Contains(t, args, "s3:bucket/repo")
}

// TestBuildPruneCommand tests CLI command for prune task (TDD - Epic 11.1)
func TestBuildPruneCommand(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "prune",
		Repository: "s3:bucket/repo",
		Retention: map[string]interface{}{
			"keepLast":  float64(7),
			"keepDaily": float64(14),
		},
	}

	cmd, args := executor.BuildCommand(task)

	assert.Equal(t, "/usr/bin/restic", cmd)
	assert.Contains(t, args, "forget")
	assert.Contains(t, args, "--prune")
	assert.Contains(t, args, "--keep-last=7")
	assert.Contains(t, args, "--keep-daily=14")
}

// TestExecuteTaskSuccess tests successful task execution (TDD - Epic 11.1)
func TestExecuteTaskSuccess(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "test-repo",
	}

	result, err := executor.Execute(task)

	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.NotEmpty(t, result.TaskID)
	assert.Greater(t, result.DurationSeconds, float64(0))
}

// TestExecuteTaskFailure tests failed task execution (TDD - Epic 11.1)
func TestExecuteTaskFailure(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/false")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "test-repo",
	}

	result, err := executor.Execute(task)

	require.NoError(t, err) // Execute should not error, but result should indicate failure
	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.TaskID)
}

// TestExecuteTaskCapturesOutput tests log capture (TDD - Epic 11.1)
func TestExecuteTaskCapturesOutput(t *testing.T) {
	executor := agent.NewTaskExecutor("echo")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "test-output",
	}

	result, err := executor.Execute(task)

	require.NoError(t, err)
	assert.Contains(t, result.Log, "test-output")
	assert.NotEmpty(t, result.Log)
}

// TestExecuteTaskMeasuresDuration tests duration measurement (TDD - Epic 11.1)
func TestExecuteTaskMeasuresDuration(t *testing.T) {
	executor := agent.NewTaskExecutor("sleep")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "0.1",
	}

	start := time.Now()
	result, err := executor.Execute(task)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.InDelta(t, elapsed.Seconds(), result.DurationSeconds, 0.5)
}

// TestBuildCommandWithExecutionParams tests execution parameters (TDD - Epic 11.2)
func TestBuildCommandWithExecutionParams(t *testing.T) {
	executor := agent.NewTaskExecutor("/usr/bin/restic")

	task := agent.Task{
		TaskID:     uuid.New().String(),
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		ExecutionParams: map[string]interface{}{
			"bandwidthLimitKbps": float64(5000),
			"parallelism":        float64(4),
		},
		IncludePaths: map[string]interface{}{
			"paths": []interface{}{"/data"},
		},
	}

	_, args := executor.BuildCommand(task)

	assert.Contains(t, args, "--limit-upload=5000")
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "local.connections=4")
}

// TestTaskResultSerialization tests JSON serialization (TDD - Epic 11.4)
func TestTaskResultSerialization(t *testing.T) {
	result := agent.TaskResult{
		TaskID:          uuid.New().String(),
		Status:          "success",
		DurationSeconds: 12.5,
		Log:             "backup completed successfully",
		SnapshotID:      "abc123def",
	}

	json, err := result.ToJSON()

	require.NoError(t, err)
	assert.Contains(t, string(json), "success")
	assert.Contains(t, string(json), "12.5")
	assert.Contains(t, string(json), "abc123def")
}
