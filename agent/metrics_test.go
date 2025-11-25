package agent_test

import (
	"errors"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
)

// TestNewLoopMetrics tests metrics initialization (TDD - Epic 9.6)
func TestNewLoopMetrics(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	if metrics.GetLoopCount() != 0 {
		t.Errorf("Expected loop count 0, got %d", metrics.GetLoopCount())
	}

	if metrics.GetTotalTasksFetched() != 0 {
		t.Errorf("Expected total tasks 0, got %d", metrics.GetTotalTasksFetched())
	}

	if metrics.GetTotalHeartbeatsSent() != 0 {
		t.Errorf("Expected total heartbeats 0, got %d", metrics.GetTotalHeartbeatsSent())
	}

	if metrics.GetTotalErrors() != 0 {
		t.Errorf("Expected total errors 0, got %d", metrics.GetTotalErrors())
	}

	if metrics.GetLastHeartbeatStatus() != "never" {
		t.Errorf("Expected heartbeat status 'never', got %s", metrics.GetLastHeartbeatStatus())
	}

	if metrics.GetLastTaskFetchStatus() != "never" {
		t.Errorf("Expected task fetch status 'never', got %s", metrics.GetLastTaskFetchStatus())
	}
}

// TestLoopCountIncrement tests loop counter (TDD - Epic 9.6)
func TestLoopCountIncrement(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.IncrementLoopCount()
	if metrics.GetLoopCount() != 1 {
		t.Errorf("Expected loop count 1, got %d", metrics.GetLoopCount())
	}

	metrics.IncrementLoopCount()
	metrics.IncrementLoopCount()
	if metrics.GetLoopCount() != 3 {
		t.Errorf("Expected loop count 3, got %d", metrics.GetLoopCount())
	}
}

// TestLastLoopTimestamp tests timestamp tracking (TDD - Epic 9.6)
func TestLastLoopTimestamp(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	metrics.IncrementLoopCount()
	after := time.Now()

	timestamp := metrics.GetLastLoopTimestamp()
	if timestamp.Before(before) || timestamp.After(after) {
		t.Error("Last loop timestamp not in expected range")
	}
}

// TestRecordTasksFetched tests task counting (TDD - Epic 9.6)
func TestRecordTasksFetched(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.RecordTasksFetched(5)
	if metrics.GetTotalTasksFetched() != 5 {
		t.Errorf("Expected 5 tasks, got %d", metrics.GetTotalTasksFetched())
	}

	if metrics.GetLastTaskFetchStatus() != "success" {
		t.Errorf("Expected status 'success', got %s", metrics.GetLastTaskFetchStatus())
	}

	metrics.RecordTasksFetched(3)
	if metrics.GetTotalTasksFetched() != 8 {
		t.Errorf("Expected 8 tasks, got %d", metrics.GetTotalTasksFetched())
	}
}

// TestRecordTasksFetchedEmpty tests empty task list (TDD - Epic 9.6)
func TestRecordTasksFetchedEmpty(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.RecordTasksFetched(0)
	if metrics.GetTotalTasksFetched() != 0 {
		t.Errorf("Expected 0 tasks, got %d", metrics.GetTotalTasksFetched())
	}

	if metrics.GetLastTaskFetchStatus() != "empty" {
		t.Errorf("Expected status 'empty', got %s", metrics.GetLastTaskFetchStatus())
	}
}

// TestRecordHeartbeatSuccess tests heartbeat tracking (TDD - Epic 9.6)
func TestRecordHeartbeatSuccess(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.RecordHeartbeatSuccess()
	if metrics.GetTotalHeartbeatsSent() != 1 {
		t.Errorf("Expected 1 heartbeat, got %d", metrics.GetTotalHeartbeatsSent())
	}

	if metrics.GetLastHeartbeatStatus() != "success" {
		t.Errorf("Expected status 'success', got %s", metrics.GetLastHeartbeatStatus())
	}

	metrics.RecordHeartbeatSuccess()
	metrics.RecordHeartbeatSuccess()
	if metrics.GetTotalHeartbeatsSent() != 3 {
		t.Errorf("Expected 3 heartbeats, got %d", metrics.GetTotalHeartbeatsSent())
	}
}

// TestRecordHeartbeatError tests heartbeat error tracking (TDD - Epic 9.6)
func TestRecordHeartbeatError(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	err := errors.New("connection timeout")
	metrics.RecordHeartbeatError(err)

	if metrics.GetHeartbeatErrors() != 1 {
		t.Errorf("Expected 1 heartbeat error, got %d", metrics.GetHeartbeatErrors())
	}

	if metrics.GetTotalErrors() != 1 {
		t.Errorf("Expected 1 total error, got %d", metrics.GetTotalErrors())
	}

	if metrics.GetLastHeartbeatStatus() != "error" {
		t.Errorf("Expected status 'error', got %s", metrics.GetLastHeartbeatStatus())
	}

	if metrics.GetLastError() != "connection timeout" {
		t.Errorf("Expected error 'connection timeout', got %s", metrics.GetLastError())
	}
}

// TestRecordTaskFetchError tests task fetch error tracking (TDD - Epic 9.6)
func TestRecordTaskFetchError(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	err := errors.New("server error")
	metrics.RecordTaskFetchError(err)

	if metrics.GetTaskFetchErrors() != 1 {
		t.Errorf("Expected 1 task fetch error, got %d", metrics.GetTaskFetchErrors())
	}

	if metrics.GetTotalErrors() != 1 {
		t.Errorf("Expected 1 total error, got %d", metrics.GetTotalErrors())
	}

	if metrics.GetLastTaskFetchStatus() != "error" {
		t.Errorf("Expected status 'error', got %s", metrics.GetLastTaskFetchStatus())
	}

	if metrics.GetLastError() != "server error" {
		t.Errorf("Expected error 'server error', got %s", metrics.GetLastError())
	}
}

// TestMultipleErrors tests error accumulation (TDD - Epic 9.6)
func TestMultipleErrors(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.RecordHeartbeatError(errors.New("error 1"))
	metrics.RecordHeartbeatError(errors.New("error 2"))
	metrics.RecordTaskFetchError(errors.New("error 3"))

	if metrics.GetHeartbeatErrors() != 2 {
		t.Errorf("Expected 2 heartbeat errors, got %d", metrics.GetHeartbeatErrors())
	}

	if metrics.GetTaskFetchErrors() != 1 {
		t.Errorf("Expected 1 task fetch error, got %d", metrics.GetTaskFetchErrors())
	}

	if metrics.GetTotalErrors() != 3 {
		t.Errorf("Expected 3 total errors, got %d", metrics.GetTotalErrors())
	}
}

// TestLastErrorTimestamp tests error timestamp tracking (TDD - Epic 9.6)
func TestLastErrorTimestamp(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	metrics.RecordHeartbeatError(errors.New("test error"))
	after := time.Now()

	timestamp := metrics.GetLastErrorTimestamp()
	if timestamp.Before(before) || timestamp.After(after) {
		t.Error("Last error timestamp not in expected range")
	}
}

// TestRecordLoopDuration tests duration tracking (TDD - Epic 9.6)
func TestRecordLoopDuration(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	// First loop
	metrics.IncrementLoopCount()
	metrics.RecordLoopDuration(100 * time.Millisecond)

	avg := metrics.GetAverageLoopDuration()
	if avg != 100*time.Millisecond {
		t.Errorf("Expected average 100ms, got %v", avg)
	}

	// Second loop
	metrics.IncrementLoopCount()
	metrics.RecordLoopDuration(200 * time.Millisecond)

	avg = metrics.GetAverageLoopDuration()
	if avg != 150*time.Millisecond {
		t.Errorf("Expected average 150ms, got %v", avg)
	}
}

// TestMetricsSnapshot tests snapshot functionality (TDD - Epic 9.6)
func TestMetricsSnapshot(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	metrics.IncrementLoopCount()
	metrics.RecordTasksFetched(5)
	metrics.RecordHeartbeatSuccess()
	metrics.RecordLoopDuration(100 * time.Millisecond)

	snapshot := metrics.GetSnapshot()

	if snapshot.LoopCount != 1 {
		t.Errorf("Expected loop count 1, got %d", snapshot.LoopCount)
	}

	if snapshot.TotalTasksFetched != 5 {
		t.Errorf("Expected 5 tasks, got %d", snapshot.TotalTasksFetched)
	}

	if snapshot.TotalHeartbeatsSent != 1 {
		t.Errorf("Expected 1 heartbeat, got %d", snapshot.TotalHeartbeatsSent)
	}

	if snapshot.AverageLoopDuration != 100*time.Millisecond {
		t.Errorf("Expected 100ms duration, got %v", snapshot.AverageLoopDuration)
	}
}

// TestMetricsConcurrentAccess tests thread-safety (TDD - Epic 9.6)
func TestMetricsConcurrentAccess(t *testing.T) {
	metrics := agent.NewLoopMetrics()

	done := make(chan bool)

	// Concurrent increments
	for i := 0; i < 100; i++ {
		go func() {
			metrics.IncrementLoopCount()
			metrics.RecordTasksFetched(1)
			metrics.RecordHeartbeatSuccess()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	if metrics.GetLoopCount() != 100 {
		t.Errorf("Expected loop count 100, got %d", metrics.GetLoopCount())
	}

	if metrics.GetTotalTasksFetched() != 100 {
		t.Errorf("Expected 100 tasks, got %d", metrics.GetTotalTasksFetched())
	}

	if metrics.GetTotalHeartbeatsSent() != 100 {
		t.Errorf("Expected 100 heartbeats, got %d", metrics.GetTotalHeartbeatsSent())
	}
}
