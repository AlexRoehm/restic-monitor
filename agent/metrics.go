package agent

import (
	"sync"
	"time"
)

// LoopMetrics tracks polling loop statistics
type LoopMetrics struct {
	mu                  sync.RWMutex
	loopCount           int64
	lastLoopTimestamp   time.Time
	totalTasksFetched   int64
	totalHeartbeatsSent int64
	totalErrors         int64
	heartbeatErrors     int64
	taskFetchErrors     int64
	lastHeartbeatStatus string
	lastTaskFetchStatus string
	lastError           string
	lastErrorTimestamp  time.Time
	totalLoopDuration   time.Duration
	averageLoopDuration time.Duration
}

// NewLoopMetrics creates a new metrics tracker
func NewLoopMetrics() *LoopMetrics {
	return &LoopMetrics{
		lastLoopTimestamp:   time.Now(),
		lastHeartbeatStatus: "never",
		lastTaskFetchStatus: "never",
	}
}

// IncrementLoopCount increments the loop iteration counter
func (m *LoopMetrics) IncrementLoopCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loopCount++
	m.lastLoopTimestamp = time.Now()
}

// RecordLoopDuration records the duration of a loop iteration
func (m *LoopMetrics) RecordLoopDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLoopDuration += duration
	if m.loopCount > 0 {
		m.averageLoopDuration = time.Duration(int64(m.totalLoopDuration) / m.loopCount)
	}
}

// RecordTasksFetched records the number of tasks fetched
func (m *LoopMetrics) RecordTasksFetched(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalTasksFetched += int64(count)
	if count > 0 {
		m.lastTaskFetchStatus = "success"
	} else {
		m.lastTaskFetchStatus = "empty"
	}
}

// RecordHeartbeatSuccess records a successful heartbeat
func (m *LoopMetrics) RecordHeartbeatSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalHeartbeatsSent++
	m.lastHeartbeatStatus = "success"
}

// RecordHeartbeatError records a failed heartbeat
func (m *LoopMetrics) RecordHeartbeatError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalErrors++
	m.heartbeatErrors++
	m.lastHeartbeatStatus = "error"
	m.lastError = err.Error()
	m.lastErrorTimestamp = time.Now()
}

// RecordTaskFetchError records a failed task fetch
func (m *LoopMetrics) RecordTaskFetchError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalErrors++
	m.taskFetchErrors++
	m.lastTaskFetchStatus = "error"
	m.lastError = err.Error()
	m.lastErrorTimestamp = time.Now()
}

// GetLoopCount returns the current loop count
func (m *LoopMetrics) GetLoopCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loopCount
}

// GetLastLoopTimestamp returns the timestamp of the last loop
func (m *LoopMetrics) GetLastLoopTimestamp() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastLoopTimestamp
}

// GetTotalTasksFetched returns the total number of tasks fetched
func (m *LoopMetrics) GetTotalTasksFetched() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalTasksFetched
}

// GetTotalHeartbeatsSent returns the total number of heartbeats sent
func (m *LoopMetrics) GetTotalHeartbeatsSent() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalHeartbeatsSent
}

// GetTotalErrors returns the total number of errors
func (m *LoopMetrics) GetTotalErrors() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalErrors
}

// GetHeartbeatErrors returns the number of heartbeat errors
func (m *LoopMetrics) GetHeartbeatErrors() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.heartbeatErrors
}

// GetTaskFetchErrors returns the number of task fetch errors
func (m *LoopMetrics) GetTaskFetchErrors() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.taskFetchErrors
}

// GetLastHeartbeatStatus returns the status of the last heartbeat
func (m *LoopMetrics) GetLastHeartbeatStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastHeartbeatStatus
}

// GetLastTaskFetchStatus returns the status of the last task fetch
func (m *LoopMetrics) GetLastTaskFetchStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastTaskFetchStatus
}

// GetLastError returns the last error message
func (m *LoopMetrics) GetLastError() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// GetLastErrorTimestamp returns the timestamp of the last error
func (m *LoopMetrics) GetLastErrorTimestamp() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErrorTimestamp
}

// GetAverageLoopDuration returns the average loop duration
func (m *LoopMetrics) GetAverageLoopDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.averageLoopDuration
}

// GetSnapshot returns a snapshot of all metrics
func (m *LoopMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MetricsSnapshot{
		LoopCount:           m.loopCount,
		LastLoopTimestamp:   m.lastLoopTimestamp,
		TotalTasksFetched:   m.totalTasksFetched,
		TotalHeartbeatsSent: m.totalHeartbeatsSent,
		TotalErrors:         m.totalErrors,
		HeartbeatErrors:     m.heartbeatErrors,
		TaskFetchErrors:     m.taskFetchErrors,
		LastHeartbeatStatus: m.lastHeartbeatStatus,
		LastTaskFetchStatus: m.lastTaskFetchStatus,
		LastError:           m.lastError,
		LastErrorTimestamp:  m.lastErrorTimestamp,
		AverageLoopDuration: m.averageLoopDuration,
	}
}

// MetricsSnapshot is a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	LoopCount           int64         `json:"loop_count"`
	LastLoopTimestamp   time.Time     `json:"last_loop_timestamp"`
	TotalTasksFetched   int64         `json:"total_tasks_fetched"`
	TotalHeartbeatsSent int64         `json:"total_heartbeats_sent"`
	TotalErrors         int64         `json:"total_errors"`
	HeartbeatErrors     int64         `json:"heartbeat_errors"`
	TaskFetchErrors     int64         `json:"task_fetch_errors"`
	LastHeartbeatStatus string        `json:"last_heartbeat_status"`
	LastTaskFetchStatus string        `json:"last_task_fetch_status"`
	LastError           string        `json:"last_error,omitempty"`
	LastErrorTimestamp  time.Time     `json:"last_error_timestamp,omitempty"`
	AverageLoopDuration time.Duration `json:"average_loop_duration"`
}
