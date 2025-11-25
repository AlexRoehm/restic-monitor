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
