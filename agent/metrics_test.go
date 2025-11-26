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

// Phase 7: Execution metrics tests

// TestExecutionMetricsRetryTracking tests retry attempt counting
func TestExecutionMetricsRetryTracking(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// Record network error retries
	metrics.RecordTaskRetry("network")
	metrics.RecordTaskRetry("network")
	
	if metrics.GetTasksRetried() != 2 {
		t.Errorf("Expected 2 retries, got %d", metrics.GetTasksRetried())
	}
	
	if metrics.GetNetworkErrors() != 2 {
		t.Errorf("Expected 2 network errors, got %d", metrics.GetNetworkErrors())
	}
	
	// Record other error types
	metrics.RecordTaskRetry("resource")
	metrics.RecordTaskRetry("auth")
	metrics.RecordTaskRetry("permanent")
	
	if metrics.GetTasksRetried() != 5 {
		t.Errorf("Expected 5 retries, got %d", metrics.GetTasksRetried())
	}
	
	if metrics.GetResourceErrors() != 1 {
		t.Errorf("Expected 1 resource error, got %d", metrics.GetResourceErrors())
	}
	
	if metrics.GetAuthErrors() != 1 {
		t.Errorf("Expected 1 auth error, got %d", metrics.GetAuthErrors())
	}
	
	if metrics.GetPermanentErrors() != 1 {
		t.Errorf("Expected 1 permanent error, got %d", metrics.GetPermanentErrors())
	}
}

// TestExecutionMetricsBackoffEvents tests backoff event tracking
func TestExecutionMetricsBackoffEvents(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	metrics.RecordBackoffEvent()
	metrics.RecordBackoffEvent()
	metrics.RecordBackoffEvent()

	if metrics.GetBackoffEvents() != 3 {
		t.Errorf("Expected 3 backoff events, got %d", metrics.GetBackoffEvents())
	}
}

// TestExecutionMetricsPermanentFailures tests permanent failure tracking
func TestExecutionMetricsPermanentFailures(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	metrics.RecordPermanentFailure()
	metrics.RecordPermanentFailure()

	if metrics.GetPermanentFailures() != 2 {
		t.Errorf("Expected 2 permanent failures, got %d", metrics.GetPermanentFailures())
	}
}

// TestExecutionMetricsTaskExhaustion tests exhausted task tracking
func TestExecutionMetricsTaskExhaustion(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	metrics.RecordTaskExhausted()
	metrics.RecordTaskExhausted()
	metrics.RecordTaskExhausted()

	if metrics.GetTasksExhausted() != 3 {
		t.Errorf("Expected 3 exhausted tasks, got %d", metrics.GetTasksExhausted())
	}
}

// TestExecutionMetricsConcurrencyLimit tests concurrency limit tracking
func TestExecutionMetricsConcurrencyLimit(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	metrics.RecordConcurrencyLimitReached()
	metrics.RecordConcurrencyLimitReached()

	if metrics.GetConcurrencyLimitReached() != 2 {
		t.Errorf("Expected 2 concurrency limit events, got %d", metrics.GetConcurrencyLimitReached())
	}
}

// TestExecutionMetricsQuotaExceeded tests quota exceeded tracking
func TestExecutionMetricsQuotaExceeded(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	metrics.RecordQuotaExceeded()
	metrics.RecordQuotaExceeded()
	metrics.RecordQuotaExceeded()

	if metrics.GetQuotaExceededEvents() != 3 {
		t.Errorf("Expected 3 quota exceeded events, got %d", metrics.GetQuotaExceededEvents())
	}
}

// TestExecutionMetricsErrorCategorization tests error category breakdown
func TestExecutionMetricsErrorCategorization(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// Record various error types
	for i := 0; i < 5; i++ {
		metrics.RecordTaskRetry("network")
	}
	for i := 0; i < 3; i++ {
		metrics.RecordTaskRetry("resource")
	}
	for i := 0; i < 2; i++ {
		metrics.RecordTaskRetry("auth")
	}
	metrics.RecordTaskRetry("permanent")

	// Verify categorization
	if metrics.GetNetworkErrors() != 5 {
		t.Errorf("Expected 5 network errors, got %d", metrics.GetNetworkErrors())
	}
	
	if metrics.GetResourceErrors() != 3 {
		t.Errorf("Expected 3 resource errors, got %d", metrics.GetResourceErrors())
	}
	
	if metrics.GetAuthErrors() != 2 {
		t.Errorf("Expected 2 auth errors, got %d", metrics.GetAuthErrors())
	}
	
	if metrics.GetPermanentErrors() != 1 {
		t.Errorf("Expected 1 permanent error, got %d", metrics.GetPermanentErrors())
	}
	
	if metrics.GetTasksRetried() != 11 {
		t.Errorf("Expected 11 total retries, got %d", metrics.GetTasksRetried())
	}
}

// TestExecutionMetricsMultipleEventTypes tests mixed event recording
func TestExecutionMetricsMultipleEventTypes(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// Simulate various failure scenarios
	metrics.RecordTaskRetry("network")
	metrics.RecordBackoffEvent()
	
	metrics.RecordTaskRetry("network")
	metrics.RecordBackoffEvent()
	
	metrics.RecordTaskRetry("network")
	metrics.RecordTaskExhausted()
	
	metrics.RecordPermanentFailure()
	metrics.RecordConcurrencyLimitReached()
	metrics.RecordQuotaExceeded()

	// Verify all counters
	if metrics.GetTasksRetried() != 3 {
		t.Errorf("Expected 3 retries, got %d", metrics.GetTasksRetried())
	}
	
	if metrics.GetBackoffEvents() != 2 {
		t.Errorf("Expected 2 backoff events, got %d", metrics.GetBackoffEvents())
	}
	
	if metrics.GetTasksExhausted() != 1 {
		t.Errorf("Expected 1 exhausted task, got %d", metrics.GetTasksExhausted())
	}
	
	if metrics.GetPermanentFailures() != 1 {
		t.Errorf("Expected 1 permanent failure, got %d", metrics.GetPermanentFailures())
	}
	
	if metrics.GetConcurrencyLimitReached() != 1 {
		t.Errorf("Expected 1 concurrency limit event, got %d", metrics.GetConcurrencyLimitReached())
	}
	
	if metrics.GetQuotaExceededEvents() != 1 {
		t.Errorf("Expected 1 quota exceeded event, got %d", metrics.GetQuotaExceededEvents())
	}
}

// TestExecutionMetricsThreadSafety tests concurrent metric updates
func TestExecutionMetricsThreadSafety(t *testing.T) {
	metrics := agent.NewExecutionMetrics()
	done := make(chan bool)

	// Concurrent metric recording
	for i := 0; i < 50; i++ {
		go func() {
			metrics.RecordTaskRetry("network")
			metrics.RecordBackoffEvent()
			metrics.RecordConcurrencyLimitReached()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	if metrics.GetTasksRetried() != 50 {
		t.Errorf("Expected 50 retries, got %d", metrics.GetTasksRetried())
	}
	
	if metrics.GetBackoffEvents() != 50 {
		t.Errorf("Expected 50 backoff events, got %d", metrics.GetBackoffEvents())
	}
	
	if metrics.GetConcurrencyLimitReached() != 50 {
		t.Errorf("Expected 50 concurrency limit events, got %d", metrics.GetConcurrencyLimitReached())
	}
}

// TestExecutionMetricsInitialState tests zero initialization
func TestExecutionMetricsInitialState(t *testing.T) {
	metrics := agent.NewExecutionMetrics()

	// All counters should be zero initially
	if metrics.GetTasksRetried() != 0 {
		t.Errorf("Expected 0 retries, got %d", metrics.GetTasksRetried())
	}
	
	if metrics.GetBackoffEvents() != 0 {
		t.Errorf("Expected 0 backoff events, got %d", metrics.GetBackoffEvents())
	}
	
	if metrics.GetPermanentFailures() != 0 {
		t.Errorf("Expected 0 permanent failures, got %d", metrics.GetPermanentFailures())
	}
	
	if metrics.GetTasksExhausted() != 0 {
		t.Errorf("Expected 0 exhausted tasks, got %d", metrics.GetTasksExhausted())
	}
	
	if metrics.GetConcurrencyLimitReached() != 0 {
		t.Errorf("Expected 0 concurrency limit events, got %d", metrics.GetConcurrencyLimitReached())
	}
	
	if metrics.GetQuotaExceededEvents() != 0 {
		t.Errorf("Expected 0 quota exceeded events, got %d", metrics.GetQuotaExceededEvents())
	}
	
	if metrics.GetNetworkErrors() != 0 {
		t.Errorf("Expected 0 network errors, got %d", metrics.GetNetworkErrors())
	}
	
	if metrics.GetResourceErrors() != 0 {
		t.Errorf("Expected 0 resource errors, got %d", metrics.GetResourceErrors())
	}
	
	if metrics.GetAuthErrors() != 0 {
		t.Errorf("Expected 0 auth errors, got %d", metrics.GetAuthErrors())
	}
	
	if metrics.GetPermanentErrors() != 0 {
		t.Errorf("Expected 0 permanent errors, got %d", metrics.GetPermanentErrors())
	}
}
