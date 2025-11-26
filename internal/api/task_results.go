package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/restic-monitor/agent"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// Default retry configuration for tasks
const (
	DefaultMaxRetries   = 3
	DefaultBaseDelay    = 5 * time.Second
	DefaultMaxDelay     = 5 * time.Minute
	DefaultJitterFactor = 0.5
)

// TaskResultRequest represents the task result payload from an agent
type TaskResultRequest struct {
	TaskID          string  `json:"taskId"`
	PolicyID        string  `json:"policyId"`
	TaskType        string  `json:"taskType"`
	Status          string  `json:"status"`
	DurationSeconds float64 `json:"durationSeconds"`
	Log             string  `json:"log"`
	SnapshotID      string  `json:"snapshotId,omitempty"`
	ErrorMessage    string  `json:"errorMessage,omitempty"`
}

// TaskResultResponse represents the response to a task result submission
type TaskResultResponse struct {
	Status string `json:"status"`
}

// handleTaskResults processes task execution results from agents
func (a *API) handleTaskResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID from URL path
	// URL format: /agents/{id}/task-results
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "agents" || pathParts[2] != "task-results" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/task-results")
		return
	}

	agentIDStr := pathParts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
		return
	}

	// Parse request body
	var req TaskResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	validationErrors := validateTaskResultRequest(req)
	if len(validationErrors) > 0 {
		details := ""
		for i, err := range validationErrors {
			if i > 0 {
				details += "; "
			}
			details += err
		}
		sendError(w, http.StatusBadRequest, "Validation failed", details)
		return
	}

	// Parse task ID and policy ID
	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid task ID", "Task ID must be a valid UUID")
		return
	}

	policyID, err := uuid.Parse(req.PolicyID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid policy ID", "Policy ID must be a valid UUID")
		return
	}

	// Verify agent exists
	var agent store.Agent
	err = a.store.GetDB().Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).First(&agent).Error
	if err != nil {
		log.Printf("Agent not found: %s (tenant: %s)", agentID, a.store.GetTenantID())
		sendError(w, http.StatusNotFound, "Agent not found", fmt.Sprintf("No agent found with ID: %s", agentID))
		return
	}

	// Create or update backup run
	now := time.Now()
	durationSeconds := req.DurationSeconds

	backupRun := store.BackupRun{
		ID:              taskID,
		TenantID:        a.store.GetTenantID(),
		AgentID:         agentID,
		PolicyID:        policyID,
		StartTime:       now.Add(-time.Duration(req.DurationSeconds) * time.Second), // Approximate start time
		EndTime:         &now,
		Status:          req.Status,
		DurationSeconds: &durationSeconds,
	}

	// Set optional fields if provided
	if req.ErrorMessage != "" {
		backupRun.ErrorMessage = &req.ErrorMessage
	}

	if req.SnapshotID != "" {
		backupRun.SnapshotID = &req.SnapshotID
	}

	// Save backup run (upsert)
	if err := a.store.UpsertBackupRun(ctx, &backupRun); err != nil {
		log.Printf("Failed to save backup run %s: %v", taskID, err)
		sendError(w, http.StatusInternalServerError, "Failed to save backup run", err.Error())
		return
	}

	// Store log content if provided (with automatic chunking for large logs)
	if req.Log != "" {
		if err := a.store.StoreBackupRunLogs(ctx, taskID, req.Log); err != nil {
			log.Printf("Failed to store logs for backup run %s: %v", taskID, err)
			// Don't fail the request - log storage is not critical
			// The backup run itself was saved successfully
		}
	}

	// Update task retry state based on result
	if err := a.updateTaskRetryState(ctx, taskID, req.Status, req.ErrorMessage); err != nil {
		log.Printf("Failed to update task retry state for %s: %v", taskID, err)
		// Don't fail the request - retry state update is not critical for the immediate response
	}

	// Log the result
	log.Printf("Task result from agent %s: task=%s, policy=%s, type=%s, status=%s, duration=%.1fs",
		agentID, taskID, policyID, req.TaskType, req.Status, req.DurationSeconds)

	// Return success response
	response := TaskResultResponse{
		Status: "ok",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// validateTaskResultRequest validates the task result request
func validateTaskResultRequest(req TaskResultRequest) []string {
	var errors []string

	if req.TaskID == "" {
		errors = append(errors, "taskId is required")
	}

	if req.PolicyID == "" {
		errors = append(errors, "policyId is required")
	}

	if req.TaskType == "" {
		errors = append(errors, "taskType is required")
	}

	if req.Status == "" {
		errors = append(errors, "status is required")
	}

	if req.DurationSeconds < 0 {
		errors = append(errors, "durationSeconds must be >= 0")
	}

	return errors
}

// updateTaskRetryState updates task retry tracking based on execution result
func (a *API) updateTaskRetryState(ctx context.Context, taskID uuid.UUID, status string, errorMessage string) error {
	// Fetch the task
	var task store.Task
	if err := a.store.GetDB().Where("id = ?", taskID).First(&task).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Initialize retry fields if not set
	if task.RetryCount == nil {
		zero := 0
		task.RetryCount = &zero
	}
	if task.MaxRetries == nil {
		maxRetries := DefaultMaxRetries
		task.MaxRetries = &maxRetries
	}

	// Handle based on status
	if status == "success" || status == "completed" {
		// Success - reset retry state
		zero := 0
		task.RetryCount = &zero
		task.NextRetryAt = nil
		task.LastErrorCategory = nil
		task.Status = "completed"
		task.CompletedAt = timePtr(time.Now())
	} else if status == "failure" || status == "failed" {
		// Check if error is permanent
		retryInfo := agent.RetryInfo{
			RetryCount:  *task.RetryCount,
			MaxRetries:  *task.MaxRetries,
			LastError:   errorMessage,
			NextRetryAt: task.NextRetryAt,
		}

		shouldRetry, reason := agent.ShouldRetryTask(retryInfo)

		if shouldRetry {
			// Increment retry count
			newRetryCount := *task.RetryCount + 1
			task.RetryCount = &newRetryCount

			// Calculate next retry time
			nextRetry := agent.CalculateNextRetryTime(
				newRetryCount,
				DefaultBaseDelay,
				DefaultMaxDelay,
				DefaultJitterFactor,
				time.Now(),
			)
			task.NextRetryAt = &nextRetry

			// Categorize error
			errorCategory := categorizeError(errorMessage)
			task.LastErrorCategory = &errorCategory

			// Keep task in pending state for retry
			task.Status = "pending"

			log.Printf("Task %s will retry (attempt %d/%d) at %s",
				taskID, newRetryCount, *task.MaxRetries, nextRetry.Format(time.RFC3339))
		} else {
			// Permanent failure or max retries exceeded
			task.Status = "failed"
			task.CompletedAt = timePtr(time.Now())

			if reason != "" {
				log.Printf("Task %s marked as failed: %s", taskID, reason)
			}
		}
	}

	// Save updated task
	if err := a.store.GetDB().Save(&task).Error; err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

// categorizeError determines error category for retry decisions
func categorizeError(errorMsg string) string {
	if errorMsg == "" {
		return "unknown"
	}

	lowerMsg := strings.ToLower(errorMsg)

	// Network errors
	if strings.Contains(lowerMsg, "timeout") ||
		strings.Contains(lowerMsg, "connection refused") ||
		strings.Contains(lowerMsg, "network") {
		return "network"
	}

	// Transient errors
	if strings.Contains(lowerMsg, "locked") ||
		strings.Contains(lowerMsg, "temporarily unavailable") {
		return "transient"
	}

	// Permission errors
	if strings.Contains(lowerMsg, "permission denied") ||
		strings.Contains(lowerMsg, "access denied") ||
		strings.Contains(lowerMsg, "forbidden") {
		return "permission"
	}

	// Repository errors
	if strings.Contains(lowerMsg, "not found") ||
		strings.Contains(lowerMsg, "invalid repository") {
		return "repository"
	}

	return "unknown"
}

// timePtr is a helper to get a pointer to a time value
func timePtr(t time.Time) *time.Time {
	return &t
}
