package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// Scheduler manages periodic task generation based on policy schedules
type Scheduler struct {
	store    *store.Store
	interval time.Duration
	running  bool
	mu       sync.RWMutex
	stopCh   chan struct{}
	doneCh   chan struct{}
	metrics  *SchedulerMetrics
}

// PolicyTaskState tracks scheduling state for a policy and task type
type PolicyTaskState struct {
	PolicyID uuid.UUID
	TaskType string
	LastRun  *time.Time
	NextRun  *time.Time
}

// NewScheduler creates a new scheduler
func NewScheduler(s *store.Store, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    s,
		interval: interval,
		running:  false,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		metrics:  NewSchedulerMetrics(),
	}
}

// Start begins the scheduler background loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	go s.loop(ctx)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetMetrics returns a snapshot of scheduler metrics
func (s *Scheduler) GetMetrics() MetricsSnapshot {
	return s.metrics.GetSnapshot()
}

// loop is the main scheduler loop
func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	if err := s.runOnce(ctx); err != nil {
		log.Printf("scheduler: error in initial run: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.runOnce(ctx); err != nil {
				log.Printf("scheduler: error in scheduled run: %v", err)
			}
		}
	}
}

// runOnce executes one scheduler iteration
func (s *Scheduler) runOnce(ctx context.Context) error {
	startTime := time.Now()

	// Get all enabled policies
	policies, err := s.store.ListPolicies(ctx, store.PolicyFilter{Enabled: boolPtr(true)})
	if err != nil {
		s.metrics.RecordError(err)
		return fmt.Errorf("failed to list policies: %w", err)
	}

	now := time.Now()
	policiesProcessed := int64(0)

	for _, policy := range policies {
		// Process backup schedule (always present)
		if err := s.processPolicyScheduleType(ctx, &policy, "backup", policy.Schedule, now); err != nil {
			log.Printf("scheduler: error processing policy %s backup schedule: %v", policy.ID, err)
			s.metrics.RecordError(err)
		} else {
			policiesProcessed++
		}

		// Process check schedule if configured
		if policy.CheckSchedule != nil && *policy.CheckSchedule != "" {
			if err := s.processPolicyScheduleType(ctx, &policy, "check", *policy.CheckSchedule, now); err != nil {
				log.Printf("scheduler: error processing policy %s check schedule: %v", policy.ID, err)
				s.metrics.RecordError(err)
			}
		}

		// Process prune schedule if configured
		if policy.PruneSchedule != nil && *policy.PruneSchedule != "" {
			if err := s.processPolicyScheduleType(ctx, &policy, "prune", *policy.PruneSchedule, now); err != nil {
				log.Printf("scheduler: error processing policy %s prune schedule: %v", policy.ID, err)
				s.metrics.RecordError(err)
			}
		}
	}

	// Record scheduler run metrics
	duration := time.Since(startTime)
	s.metrics.RecordSchedulerRun(duration, policiesProcessed)

	return nil
}

// processPolicyScheduleType handles scheduling for a single policy and task type
func (s *Scheduler) processPolicyScheduleType(ctx context.Context, policy *store.Policy, taskType string, scheduleStr string, now time.Time) error {
	// Skip disabled policies
	if !policy.Enabled {
		return nil
	}

	// Parse schedule
	parsed, err := ParseSchedule(scheduleStr)
	if err != nil {
		return fmt.Errorf("invalid schedule format: %w", err)
	}

	normalized := NormalizeSchedule(parsed)

	// Get current scheduling state
	state, err := s.GetPolicyTaskState(ctx, policy.ID, taskType)
	if err != nil {
		return fmt.Errorf("failed to get task state: %w", err)
	}

	var nextRun time.Time
	if state == nil || state.NextRun == nil {
		// First time scheduling
		// For interval schedules, trigger immediately on first run
		// For cron schedules, compute next occurrence
		if normalized.Type == ScheduleTypeInterval {
			// Generate task immediately on first run
			nextRun = now
		} else {
			// Cron: compute next run
			nextRun, err = ComputeNextRun(normalized, now)
			if err != nil {
				return fmt.Errorf("failed to compute next run: %w", err)
			}
		}

		// Save initial state
		if err := s.SavePolicyTaskState(ctx, policy.ID, taskType, nil, &nextRun); err != nil {
			return fmt.Errorf("failed to save initial state: %w", err)
		}

		// Fall through to task generation check below
	}

	// Reload state to get the current next_run value
	state, err = s.GetPolicyTaskState(ctx, policy.ID, taskType)
	if err != nil {
		return fmt.Errorf("failed to reload task state: %w", err)
	}

	// Check if task is due
	if state == nil || state.NextRun == nil || !now.After(*state.NextRun) && !now.Equal(*state.NextRun) {
		return nil
	}

	// Generate tasks for all agents assigned to this policy
	agents, err := s.store.GetPolicyAgents(ctx, policy.ID)
	if err != nil {
		return fmt.Errorf("failed to get policy agents: %w", err)
	}

	tasksCreated := 0
	for _, agent := range agents {
		task := &store.Task{
			AgentID:      agent.ID,
			TaskType:     taskType,
			PolicyID:     policy.ID,
			Status:       "pending",
			Repository:   policy.RepositoryURL,
			IncludePaths: policy.IncludePaths,
			ExcludePaths: policy.ExcludePaths,
			Retention:    policy.RetentionRules,
			MaxRetries:   policy.MaxRetries, // EPIC 15 Phase 5: inherit retry budget from policy
		}

		if err := s.store.CreateTask(ctx, task); err != nil {
			log.Printf("scheduler: failed to create task for agent %s: %v", agent.ID, err)
			s.metrics.RecordError(err)
			continue
		}
		tasksCreated++
	}

	// Record task generation metrics
	for i := 0; i < tasksCreated; i++ {
		s.metrics.RecordTaskGenerated(taskType)
	}

	// Update state with last run and compute next
	lastRun := now
	nextRun, err = ComputeNextRunWithLast(normalized, now, &lastRun)
	if err != nil {
		return fmt.Errorf("failed to compute next run: %w", err)
	}

	if err := s.SavePolicyTaskState(ctx, policy.ID, taskType, &lastRun, &nextRun); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Update next run in metrics
	s.metrics.UpdateNextRun(policy.ID, taskType, nextRun)

	return nil
}

// GetPolicyTaskState retrieves scheduling state for a policy and task type
func (s *Scheduler) GetPolicyTaskState(ctx context.Context, policyID uuid.UUID, taskType string) (*PolicyTaskState, error) {
	var state struct {
		PolicyID uuid.UUID
		TaskType string
		LastRun  *time.Time
		NextRun  *time.Time
	}

	err := s.store.GetDB().WithContext(ctx).
		Table("policy_task_states").
		Where("policy_id = ? AND task_type = ?", policyID, taskType).
		First(&state).Error

	if err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, err
	}

	return &PolicyTaskState{
		PolicyID: state.PolicyID,
		TaskType: state.TaskType,
		LastRun:  state.LastRun,
		NextRun:  state.NextRun,
	}, nil
}

// SavePolicyTaskState saves scheduling state for a policy and task type
func (s *Scheduler) SavePolicyTaskState(ctx context.Context, policyID uuid.UUID, taskType string, lastRun, nextRun *time.Time) error {
	// Try to update existing record first
	result := s.store.GetDB().WithContext(ctx).
		Table("policy_task_states").
		Where("policy_id = ? AND task_type = ?", policyID, taskType).
		Updates(map[string]interface{}{
			"last_run": lastRun,
			"next_run": nextRun,
		})

	if result.Error != nil {
		return result.Error
	}

	// If no rows were updated, insert a new record
	if result.RowsAffected == 0 {
		return s.store.GetDB().WithContext(ctx).
			Exec(`INSERT INTO policy_task_states (policy_id, task_type, last_run, next_run, created_at, updated_at) 
			      VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
				policyID, taskType, lastRun, nextRun).Error
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
