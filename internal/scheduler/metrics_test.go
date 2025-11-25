package scheduler

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSchedulerMetrics(t *testing.T) {
	metrics := NewSchedulerMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to be created")
	}

	if metrics.tasksGeneratedByType == nil {
		t.Error("Expected tasksGeneratedByType map to be initialized")
	}

	if metrics.nextRunsByPolicy == nil {
		t.Error("Expected nextRunsByPolicy map to be initialized")
	}

	if metrics.lastRunTimestamp.IsZero() {
		t.Error("Expected lastRunTimestamp to be initialized")
	}
}

func TestRecordTaskGenerated(t *testing.T) {
	metrics := NewSchedulerMetrics()

	metrics.RecordTaskGenerated("backup")
	metrics.RecordTaskGenerated("backup")
	metrics.RecordTaskGenerated("check")

	if metrics.GetTasksGeneratedTotal() != 3 {
		t.Errorf("Expected 3 total tasks, got %d", metrics.GetTasksGeneratedTotal())
	}

	snapshot := metrics.GetSnapshot()
	if snapshot.TasksGeneratedByType["backup"] != 2 {
		t.Errorf("Expected 2 backup tasks, got %d", snapshot.TasksGeneratedByType["backup"])
	}
	if snapshot.TasksGeneratedByType["check"] != 1 {
		t.Errorf("Expected 1 check task, got %d", snapshot.TasksGeneratedByType["check"])
	}
}

func TestRecordError(t *testing.T) {
	metrics := NewSchedulerMetrics()

	err := fmt.Errorf("test error")
	metrics.RecordError(err)

	if metrics.GetErrorsTotal() != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.GetErrorsTotal())
	}

	snapshot := metrics.GetSnapshot()
	if snapshot.LastError != "test error" {
		t.Errorf("Expected last error to be 'test error', got '%s'", snapshot.LastError)
	}
	if snapshot.LastErrorTimestamp.IsZero() {
		t.Error("Expected last error timestamp to be set")
	}
}

func TestRecordSchedulerRun(t *testing.T) {
	metrics := NewSchedulerMetrics()

	duration1 := 100 * time.Millisecond
	duration2 := 200 * time.Millisecond

	metrics.RecordSchedulerRun(duration1, 5)
	time.Sleep(10 * time.Millisecond) // Ensure time progresses
	metrics.RecordSchedulerRun(duration2, 3)

	snapshot := metrics.GetSnapshot()

	if snapshot.TotalRuns != 2 {
		t.Errorf("Expected 2 total runs, got %d", snapshot.TotalRuns)
	}

	if snapshot.PoliciesProcessed != 8 {
		t.Errorf("Expected 8 policies processed, got %d", snapshot.PoliciesProcessed)
	}

	expectedAverage := (duration1 + duration2) / 2
	if snapshot.AverageProcessingTime != expectedAverage {
		t.Errorf("Expected average processing time %v, got %v", expectedAverage, snapshot.AverageProcessingTime)
	}

	if snapshot.LastRunTimestamp.IsZero() {
		t.Error("Expected last run timestamp to be set")
	}
}

func TestUpdateNextRun(t *testing.T) {
	metrics := NewSchedulerMetrics()

	policyID := uuid.New()
	nextRun := time.Now().Add(1 * time.Hour)

	metrics.UpdateNextRun(policyID, "backup", nextRun)

	seconds := metrics.GetNextRunSeconds(policyID, "backup")
	if seconds < 3590 || seconds > 3610 { // Allow 10 second variance
		t.Errorf("Expected ~3600 seconds until next run, got %.2f", seconds)
	}

	snapshot := metrics.GetSnapshot()
	if len(snapshot.NextRunsByPolicy) != 1 {
		t.Errorf("Expected 1 policy in next runs, got %d", len(snapshot.NextRunsByPolicy))
	}

	taskTypes := snapshot.NextRunsByPolicy[policyID]
	if len(taskTypes) != 1 {
		t.Errorf("Expected 1 task type for policy, got %d", len(taskTypes))
	}

	storedNextRun := taskTypes["backup"]
	if !storedNextRun.Equal(nextRun) {
		t.Errorf("Expected next run %v, got %v", nextRun, storedNextRun)
	}
}

func TestUpdateMultipleNextRuns(t *testing.T) {
	metrics := NewSchedulerMetrics()

	policy1 := uuid.New()
	policy2 := uuid.New()

	metrics.UpdateNextRun(policy1, "backup", time.Now().Add(1*time.Hour))
	metrics.UpdateNextRun(policy1, "check", time.Now().Add(2*time.Hour))
	metrics.UpdateNextRun(policy2, "prune", time.Now().Add(3*time.Hour))

	snapshot := metrics.GetSnapshot()

	if len(snapshot.NextRunsByPolicy) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(snapshot.NextRunsByPolicy))
	}

	if len(snapshot.NextRunsByPolicy[policy1]) != 2 {
		t.Errorf("Expected 2 task types for policy1, got %d", len(snapshot.NextRunsByPolicy[policy1]))
	}

	if len(snapshot.NextRunsByPolicy[policy2]) != 1 {
		t.Errorf("Expected 1 task type for policy2, got %d", len(snapshot.NextRunsByPolicy[policy2]))
	}
}

func TestGetNextRunSecondsNonExistent(t *testing.T) {
	metrics := NewSchedulerMetrics()

	policyID := uuid.New()
	seconds := metrics.GetNextRunSeconds(policyID, "backup")

	if seconds != 0 {
		t.Errorf("Expected 0 seconds for non-existent policy, got %.2f", seconds)
	}
}

func TestMetricsConcurrency(t *testing.T) {
	metrics := NewSchedulerMetrics()
	iterations := 1000

	var wg sync.WaitGroup
	wg.Add(4)

	// Concurrent task generation
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			metrics.RecordTaskGenerated("backup")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			metrics.RecordTaskGenerated("check")
		}
	}()

	// Concurrent error recording
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			metrics.RecordError(fmt.Errorf("error %d", i))
		}
	}()

	// Concurrent snapshot reads
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = metrics.GetSnapshot()
		}
	}()

	wg.Wait()

	snapshot := metrics.GetSnapshot()

	if snapshot.TasksGeneratedTotal != int64(2*iterations) {
		t.Errorf("Expected %d tasks, got %d", 2*iterations, snapshot.TasksGeneratedTotal)
	}

	if snapshot.ErrorsTotal != int64(iterations) {
		t.Errorf("Expected %d errors, got %d", iterations, snapshot.ErrorsTotal)
	}

	if snapshot.TasksGeneratedByType["backup"] != int64(iterations) {
		t.Errorf("Expected %d backup tasks, got %d", iterations, snapshot.TasksGeneratedByType["backup"])
	}

	if snapshot.TasksGeneratedByType["check"] != int64(iterations) {
		t.Errorf("Expected %d check tasks, got %d", iterations, snapshot.TasksGeneratedByType["check"])
	}
}

func TestGetSnapshot(t *testing.T) {
	metrics := NewSchedulerMetrics()

	// Record some data
	metrics.RecordTaskGenerated("backup")
	metrics.RecordTaskGenerated("check")
	metrics.RecordError(fmt.Errorf("test error"))
	metrics.RecordSchedulerRun(100*time.Millisecond, 5)

	policyID := uuid.New()
	metrics.UpdateNextRun(policyID, "backup", time.Now().Add(1*time.Hour))

	snapshot := metrics.GetSnapshot()

	// Verify snapshot is independent
	metrics.RecordTaskGenerated("prune")

	if snapshot.TasksGeneratedTotal != 2 {
		t.Errorf("Snapshot should be independent, expected 2 tasks, got %d", snapshot.TasksGeneratedTotal)
	}

	if metrics.GetTasksGeneratedTotal() != 3 {
		t.Errorf("Metrics should have 3 tasks after additional record, got %d", metrics.GetTasksGeneratedTotal())
	}
}

func TestAverageProcessingTime(t *testing.T) {
	metrics := NewSchedulerMetrics()

	// Record three runs with different durations
	metrics.RecordSchedulerRun(100*time.Millisecond, 1)
	metrics.RecordSchedulerRun(200*time.Millisecond, 1)
	metrics.RecordSchedulerRun(300*time.Millisecond, 1)

	snapshot := metrics.GetSnapshot()

	expected := 200 * time.Millisecond
	if snapshot.AverageProcessingTime != expected {
		t.Errorf("Expected average processing time %v, got %v", expected, snapshot.AverageProcessingTime)
	}
}
