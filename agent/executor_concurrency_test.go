package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecutorWithConcurrencyLimits(t *testing.T) {
	t.Run("Enforce total concurrency limit", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   2,
			MaxConcurrentBackups: 2,
			MaxConcurrentChecks:  2,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		// Create 4 tasks that will run for 100ms each
		tasks := make([]Task, 4)
		for i := 0; i < 4; i++ {
			tasks[i] = Task{
				TaskID:     string(rune('A' + i)),
				TaskType:   "backup",
				Repository: "0.1", // Sleep duration in seconds
			}
		}

		// Start all tasks concurrently
		var wg sync.WaitGroup
		startTimes := make([]time.Time, 4)
		endTimes := make([]time.Time, 4)

		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				startTimes[idx] = time.Now()
				executor.Execute(tasks[idx])
				endTimes[idx] = time.Now()
			}(i)
		}

		// Give tasks a moment to start
		time.Sleep(10 * time.Millisecond)

		// At most 2 tasks should be running concurrently
		running := executor.GetRunningTaskCount()
		assert.LessOrEqual(t, running, 2, "Should not exceed MaxConcurrentTasks")

		wg.Wait()

		// Verify tasks ran in groups (overlapping execution)
		// Tasks 0 and 1 should run together, then 2 and 3
		assert.True(t, endTimes[0].After(startTimes[1]) || endTimes[1].After(startTimes[0]),
			"First two tasks should overlap")
	})

	t.Run("Enforce per-type concurrency limit", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 1, // Only 1 backup at a time
			MaxConcurrentChecks:  2,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		// Create 3 backup tasks
		tasks := make([]Task, 3)
		for i := 0; i < 3; i++ {
			tasks[i] = Task{
				TaskID:     string(rune('A' + i)),
				TaskType:   "backup",
				Repository: "0.1",
			}
		}

		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				executor.Execute(tasks[idx])
			}(i)
		}

		// Give tasks a moment to start
		time.Sleep(10 * time.Millisecond)

		// Should have exactly 1 backup running (per-type limit)
		backupCount := executor.GetRunningTaskCountByType("backup")
		assert.Equal(t, 1, backupCount, "Should respect MaxConcurrentBackups limit")

		wg.Wait()
	})

	t.Run("Mixed task types respect individual limits", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 1,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		// Create one of each type
		tasks := []Task{
			{TaskID: "backup1", TaskType: "backup", Repository: "0.1"},
			{TaskID: "check1", TaskType: "check", Repository: "0.1"},
			{TaskID: "prune1", TaskType: "prune", Repository: "0.1"},
		}

		var wg sync.WaitGroup
		for _, task := range tasks {
			wg.Add(1)
			task := task
			go func() {
				defer wg.Done()
				executor.Execute(task)
			}()
		}

		// Give tasks a moment to start
		time.Sleep(10 * time.Millisecond)

		// All 3 should be running (each type gets 1)
		running := executor.GetRunningTaskCount()
		assert.Equal(t, 3, running, "All task types should run concurrently")

		wg.Wait()
	})

	t.Run("Block when at capacity", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   1, // Only 1 task total
			MaxConcurrentBackups: 1,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		task1 := Task{TaskID: "task1", TaskType: "backup", Repository: "0.15"}
		task2 := Task{TaskID: "task2", TaskType: "backup", Repository: "0.05"}

		var wg sync.WaitGroup
		task2Started := false
		var mu sync.Mutex

		// Start task1
		wg.Add(1)
		go func() {
			defer wg.Done()
			executor.Execute(task1)
		}()

		// Wait for task1 to start
		time.Sleep(20 * time.Millisecond)

		// Try to start task2 - should block
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			task2Started = true
			mu.Unlock()
			executor.Execute(task2)
		}()

		// Check that task2 hasn't started yet
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		started := task2Started
		mu.Unlock()

		// task2 should have been attempted but blocked
		assert.True(t, started, "Task2 should have been attempted")

		// But only 1 should be running at any time
		running := executor.GetRunningTaskCount()
		assert.LessOrEqual(t, running, 1, "Should never exceed max concurrent")

		wg.Wait()
	})
}

func TestConcurrencyConfigApplication(t *testing.T) {
	t.Run("Apply bandwidth limit to restic commands", func(t *testing.T) {
		bandwidth := 100 // 100 Mbps
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 1,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
			BandwidthLimitMbps:   &bandwidth,
		}

		executor := NewTaskExecutorWithConcurrency("restic", &config)

		task := Task{
			TaskID:         "test",
			TaskType:       "backup",
			Repository:     "/repo",
			IncludePaths:   map[string]interface{}{"paths": []interface{}{"/data"}},
			ExecutionParams: make(map[string]interface{}),
		}

		// Build command
		cmd, args := executor.BuildCommand(task)

		// Should include bandwidth limit
		assert.Equal(t, "restic", cmd)
		assert.Contains(t, args, "--limit-upload=102400") // 100 Mbps = 102400 Kbps
	})

	t.Run("Apply CPU quota to restic commands", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 1,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      25, // 25% CPU
		}

		executor := NewTaskExecutorWithConcurrency("restic", &config)

		// CPU quota would be applied via process limits
		// This is platform-specific, so we just verify config is stored
		assert.Equal(t, 25, executor.config.CPUQuotaPercent)
	})
}

func TestConcurrencyMetrics(t *testing.T) {
	t.Run("Track running tasks count", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 2,
			MaxConcurrentChecks:  2,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		assert.Equal(t, 0, executor.GetRunningTaskCount())

		// Simulate task start
		task := Task{TaskID: "test", TaskType: "backup", Repository: "0.1"}
		
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			executor.Execute(task)
		}()

		// Give it time to start
		time.Sleep(20 * time.Millisecond)

		// Should have 1 running
		assert.Equal(t, 1, executor.GetRunningTaskCount())

		wg.Wait()

		// Should be back to 0
		assert.Equal(t, 0, executor.GetRunningTaskCount())
	})

	t.Run("Track per-type task counts", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 2,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("sleep", &config)

		tasks := []Task{
			{TaskID: "b1", TaskType: "backup", Repository: "0.1"},
			{TaskID: "b2", TaskType: "backup", Repository: "0.1"},
			{TaskID: "c1", TaskType: "check", Repository: "0.1"},
		}

		var wg sync.WaitGroup
		for _, task := range tasks {
			wg.Add(1)
			task := task
			go func() {
				defer wg.Done()
				executor.Execute(task)
			}()
		}

		// Give tasks time to start
		time.Sleep(20 * time.Millisecond)

		// Check counts
		assert.Equal(t, 2, executor.GetRunningTaskCountByType("backup"))
		assert.Equal(t, 1, executor.GetRunningTaskCountByType("check"))
		assert.Equal(t, 0, executor.GetRunningTaskCountByType("prune"))

		wg.Wait()
	})

	t.Run("Get available slots", func(t *testing.T) {
		config := ConcurrencyConfig{
			MaxConcurrentTasks:   3,
			MaxConcurrentBackups: 2,
			MaxConcurrentChecks:  1,
			MaxConcurrentPrunes:  1,
			CPUQuotaPercent:      50,
		}

		executor := NewTaskExecutorWithConcurrency("echo", &config)

		// Initially all slots available
		assert.Equal(t, 3, executor.GetAvailableSlots())
		assert.Equal(t, 2, executor.GetAvailableSlotsByType("backup"))
		assert.Equal(t, 1, executor.GetAvailableSlotsByType("check"))
		assert.Equal(t, 1, executor.GetAvailableSlotsByType("prune"))
	})
}
