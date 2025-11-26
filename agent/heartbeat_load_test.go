package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeartbeatPayloadWithLoad(t *testing.T) {
	t.Run("Include current task count", func(t *testing.T) {
		config := &ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 2,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("echo", config)
		
		// Simulate running tasks
		executor.mu.Lock()
		executor.runningTasks["task1"] = "backup"
		executor.runningTasks["task2"] = "check"
		executor.mu.Unlock()

		payload := BuildHeartbeatPayloadWithLoad(executor, "1.0.0", 100)

		assert.NotNil(t, payload.CurrentTasksCount)
		assert.Equal(t, 2, *payload.CurrentTasksCount)
		assert.NotNil(t, payload.RunningTaskTypes)
		assert.Contains(t, payload.RunningTaskTypes, TaskTypeCount{TaskType: "backup", Count: 1})
		assert.Contains(t, payload.RunningTaskTypes, TaskTypeCount{TaskType: "check", Count: 1})
	})

	t.Run("Include available capacity", func(t *testing.T) {
		config := &ConcurrencyConfig{
			MaxConcurrentTasks:   5,
			MaxConcurrentBackups: 3,
			MaxConcurrentChecks:  2,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      75,
		}

		executor := NewTaskExecutorWithConcurrency("echo", config)
		
		// Simulate 2 running tasks
		executor.mu.Lock()
		executor.runningTasks["task1"] = "backup"
		executor.runningTasks["task2"] = "backup"
		executor.mu.Unlock()

		// Acquire slots to reflect reality
		executor.totalLimiter.semaphore <- struct{}{}
		executor.totalLimiter.semaphore <- struct{}{}
		executor.typeLimiters["backup"].semaphore <- struct{}{}
		executor.typeLimiters["backup"].semaphore <- struct{}{}

		payload := BuildHeartbeatPayloadWithLoad(executor, "1.0.0", 100)

		assert.NotNil(t, payload.AvailableSlots)
		assert.Equal(t, 3, *payload.AvailableSlots) // 5 total - 2 used
		assert.NotNil(t, payload.AvailableSlotsByType)
		
		backupSlots := findTaskTypeCapacity(payload.AvailableSlotsByType, "backup")
		assert.NotNil(t, backupSlots)
		assert.Equal(t, 1, backupSlots.Available) // 3 backup - 2 used
		
		checkSlots := findTaskTypeCapacity(payload.AvailableSlotsByType, "check")
		assert.NotNil(t, checkSlots)
		assert.Equal(t, 2, checkSlots.Available) // 2 check - 0 used
	})

	t.Run("Handle no running tasks", func(t *testing.T) {
		config := &ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 1,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("echo", config)

		payload := BuildHeartbeatPayloadWithLoad(executor, "1.0.0", 100)

		assert.NotNil(t, payload.CurrentTasksCount)
		assert.Equal(t, 0, *payload.CurrentTasksCount)
		assert.NotNil(t, payload.RunningTaskTypes)
		assert.Empty(t, payload.RunningTaskTypes)
		assert.NotNil(t, payload.AvailableSlots)
		assert.Equal(t, 3, *payload.AvailableSlots)
	})

	t.Run("Executor without concurrency config", func(t *testing.T) {
		executor := NewTaskExecutor("echo")

		payload := BuildHeartbeatPayloadWithLoad(executor, "1.0.0", 100)

		// Should not include concurrency fields
		assert.Nil(t, payload.CurrentTasksCount)
		assert.Nil(t, payload.RunningTaskTypes)
		assert.Nil(t, payload.AvailableSlots)
		assert.Nil(t, payload.AvailableSlotsByType)
	})
}

// Helper function to find task type capacity
func findTaskTypeCapacity(capacities []TaskTypeCapacity, taskType string) *TaskTypeCapacity {
	for _, c := range capacities {
		if c.TaskType == taskType {
			return &c
		}
	}
	return nil
}

// Helper function to find task type count
func findTaskTypeCount(counts []TaskTypeCount, taskType string) *TaskTypeCount {
	for _, c := range counts {
		if c.TaskType == taskType {
			return &c
		}
	}
	return nil
}
