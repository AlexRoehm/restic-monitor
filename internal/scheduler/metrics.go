package scheduler

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// SchedulerMetrics tracks scheduler statistics
type SchedulerMetrics struct {
	mu                      sync.RWMutex
	tasksGeneratedTotal     int64
	tasksGeneratedByType    map[string]int64 // backup, check, prune
	lastRunTimestamp        time.Time
	totalRuns               int64
	errorsTotal             int64
	lastError               string
	lastErrorTimestamp      time.Time
	policiesProcessed       int64
	nextRunsByPolicy        map[uuid.UUID]map[string]time.Time // policy_id -> task_type -> next_run
	averageProcessingTime   time.Duration
	totalProcessingDuration time.Duration
}

// NewSchedulerMetrics creates a new metrics tracker
func NewSchedulerMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{
		tasksGeneratedByType: make(map[string]int64),
		nextRunsByPolicy:     make(map[uuid.UUID]map[string]time.Time),
		lastRunTimestamp:     time.Now(),
	}
}

// RecordSchedulerRun records a scheduler iteration
func (m *SchedulerMetrics) RecordSchedulerRun(duration time.Duration, policiesProcessed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRuns++
	m.lastRunTimestamp = time.Now()
	m.policiesProcessed += policiesProcessed
	m.totalProcessingDuration += duration
	if m.totalRuns > 0 {
		m.averageProcessingTime = time.Duration(int64(m.totalProcessingDuration) / m.totalRuns)
	}
}

// RecordTaskGenerated records a task generation
func (m *SchedulerMetrics) RecordTaskGenerated(taskType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasksGeneratedTotal++
	m.tasksGeneratedByType[taskType]++
}

// RecordError records an error
func (m *SchedulerMetrics) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorsTotal++
	m.lastError = err.Error()
	m.lastErrorTimestamp = time.Now()
}

// UpdateNextRun updates the next run time for a policy and task type
func (m *SchedulerMetrics) UpdateNextRun(policyID uuid.UUID, taskType string, nextRun time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nextRunsByPolicy[policyID] == nil {
		m.nextRunsByPolicy[policyID] = make(map[string]time.Time)
	}
	m.nextRunsByPolicy[policyID][taskType] = nextRun
}

// GetSnapshot returns a snapshot of current metrics
func (m *SchedulerMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Copy task counts
	tasksByType := make(map[string]int64)
	for k, v := range m.tasksGeneratedByType {
		tasksByType[k] = v
	}

	// Copy next runs
	nextRuns := make(map[uuid.UUID]map[string]time.Time)
	for policyID, taskTypes := range m.nextRunsByPolicy {
		nextRuns[policyID] = make(map[string]time.Time)
		for taskType, nextRun := range taskTypes {
			nextRuns[policyID][taskType] = nextRun
		}
	}

	return MetricsSnapshot{
		TasksGeneratedTotal:   m.tasksGeneratedTotal,
		TasksGeneratedByType:  tasksByType,
		LastRunTimestamp:      m.lastRunTimestamp,
		TotalRuns:             m.totalRuns,
		ErrorsTotal:           m.errorsTotal,
		LastError:             m.lastError,
		LastErrorTimestamp:    m.lastErrorTimestamp,
		PoliciesProcessed:     m.policiesProcessed,
		NextRunsByPolicy:      nextRuns,
		AverageProcessingTime: m.averageProcessingTime,
	}
}

// GetTasksGeneratedTotal returns the total number of tasks generated
func (m *SchedulerMetrics) GetTasksGeneratedTotal() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasksGeneratedTotal
}

// GetLastRunTimestamp returns the last run timestamp
func (m *SchedulerMetrics) GetLastRunTimestamp() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastRunTimestamp
}

// GetErrorsTotal returns the total number of errors
func (m *SchedulerMetrics) GetErrorsTotal() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorsTotal
}

// GetNextRunSeconds returns seconds until the next scheduled run for a policy and task type
func (m *SchedulerMetrics) GetNextRunSeconds(policyID uuid.UUID, taskType string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if taskTypes, ok := m.nextRunsByPolicy[policyID]; ok {
		if nextRun, ok := taskTypes[taskType]; ok {
			return time.Until(nextRun).Seconds()
		}
	}
	return 0
}

// MetricsSnapshot represents a point-in-time snapshot of scheduler metrics
type MetricsSnapshot struct {
	TasksGeneratedTotal   int64                              `json:"tasks_generated_total"`
	TasksGeneratedByType  map[string]int64                   `json:"tasks_generated_by_type"`
	LastRunTimestamp      time.Time                          `json:"last_run_timestamp"`
	TotalRuns             int64                              `json:"total_runs"`
	ErrorsTotal           int64                              `json:"errors_total"`
	LastError             string                             `json:"last_error,omitempty"`
	LastErrorTimestamp    time.Time                          `json:"last_error_timestamp,omitempty"`
	PoliciesProcessed     int64                              `json:"policies_processed"`
	NextRunsByPolicy      map[uuid.UUID]map[string]time.Time `json:"next_runs_by_policy"`
	AverageProcessingTime time.Duration                      `json:"average_processing_time"`
}
