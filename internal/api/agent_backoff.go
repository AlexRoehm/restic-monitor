package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// AgentBackoffResponse represents the backoff state of an agent
type AgentBackoffResponse struct {
	AgentID          uuid.UUID         `json:"agent_id"`
	Hostname         string            `json:"hostname"`
	TasksInBackoff   int               `json:"tasks_in_backoff"`
	EarliestRetryAt  *time.Time        `json:"earliest_retry_at,omitempty"`
	BackoffTasks     []BackoffTaskInfo `json:"backoff_tasks"`
	LastUpdatedAt    time.Time         `json:"last_updated_at"`
}

// BackoffTaskInfo represents a task in backoff state
type BackoffTaskInfo struct {
	TaskID          uuid.UUID  `json:"task_id"`
	TaskType        string     `json:"task_type"`
	RetryCount      int        `json:"retry_count"`
	MaxRetries      int        `json:"max_retries"`
	NextRetryAt     time.Time  `json:"next_retry_at"`
	ErrorCategory   string     `json:"error_category,omitempty"`
}

// handleGetAgentBackoff returns backoff state for an agent
// GET /agents/{id}/backoff-status
func (a *API) handleGetAgentBackoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "agents" || pathParts[2] != "backoff-status" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/backoff-status")
		return
	}

	agentIDStr := pathParts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
		return
	}

	// Fetch agent
	var agent store.Agent
	err = a.store.GetDB().Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).First(&agent).Error
	if err != nil {
		log.Printf("Agent not found: %s (tenant: %s)", agentID, a.store.GetTenantID())
		sendError(w, http.StatusNotFound, "Agent not found", fmt.Sprintf("No agent found with ID: %s", agentID))
		return
	}

	// Get tasks in backoff for this agent
	var tasks []store.Task
	now := time.Now()
	err = a.store.GetDB().WithContext(ctx).
		Where("agent_id = ? AND status = ? AND next_retry_at > ?", agentID, "pending", now).
		Order("next_retry_at ASC").
		Find(&tasks).Error
	if err != nil {
		log.Printf("Failed to fetch backoff tasks for agent %s: %v", agentID, err)
		sendError(w, http.StatusInternalServerError, "Failed to fetch backoff state", err.Error())
		return
	}

	// Build response
	backoffTasks := make([]BackoffTaskInfo, 0, len(tasks))
	var earliestRetry *time.Time
	
	for _, task := range tasks {
		if task.NextRetryAt != nil {
			taskInfo := BackoffTaskInfo{
				TaskID:     task.ID,
				TaskType:   task.TaskType,
				RetryCount: safeIntValue(task.RetryCount),
				MaxRetries: safeIntValue(task.MaxRetries),
				NextRetryAt: *task.NextRetryAt,
			}
			if task.LastErrorCategory != nil {
				taskInfo.ErrorCategory = *task.LastErrorCategory
			}
			backoffTasks = append(backoffTasks, taskInfo)

			// Track earliest retry time
			if earliestRetry == nil || task.NextRetryAt.Before(*earliestRetry) {
				earliestRetry = task.NextRetryAt
			}
		}
	}

	response := AgentBackoffResponse{
		AgentID:         agentID,
		Hostname:        agent.Hostname,
		TasksInBackoff:  len(backoffTasks),
		EarliestRetryAt: earliestRetry,
		BackoffTasks:    backoffTasks,
		LastUpdatedAt:   time.Now(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateAgentBackoffState calculates and updates agent backoff state based on tasks
func (a *API) UpdateAgentBackoffState(agentID uuid.UUID) error {
	ctx := a.store.GetDB().Statement.Context
	now := time.Now()

	// Count tasks in backoff (pending with next_retry_at in future)
	var tasksInBackoff int64
	err := a.store.GetDB().WithContext(ctx).Model(&store.Task{}).
		Where("agent_id = ? AND status = ? AND next_retry_at > ?", agentID, "pending", now).
		Count(&tasksInBackoff).Error
	if err != nil {
		return fmt.Errorf("failed to count backoff tasks: %w", err)
	}

	// Find earliest retry time
	var earliestTask store.Task
	err = a.store.GetDB().WithContext(ctx).
		Where("agent_id = ? AND status = ? AND next_retry_at > ?", agentID, "pending", now).
		Order("next_retry_at ASC").
		First(&earliestTask).Error
	
	var earliestRetryAt *time.Time
	if err == nil && earliestTask.NextRetryAt != nil {
		earliestRetryAt = earliestTask.NextRetryAt
	}

	// Update agent backoff state
	tasksCount := int(tasksInBackoff)
	err = a.store.GetDB().Model(&store.Agent{}).
		Where("id = ?", agentID).
		Updates(map[string]interface{}{
			"tasks_in_backoff": tasksCount,
			"earliest_retry_at": earliestRetryAt,
		}).Error
	
	if err != nil {
		return fmt.Errorf("failed to update agent backoff state: %w", err)
	}

	log.Printf("Updated agent %s backoff state: %d tasks, earliest retry: %v", 
		agentID, tasksCount, earliestRetryAt)
	
	return nil
}

// safeIntValue returns the int value or 0 if nil
func safeIntValue(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}
