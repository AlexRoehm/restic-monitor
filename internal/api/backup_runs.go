package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// BackupRunsResponse represents a list of backup runs with pagination
type BackupRunsResponse struct {
	Runs  []BackupRunSummary `json:"runs"`
	Total int                `json:"total"`
	Limit int                `json:"limit,omitempty"`
}

// BackupRunSummary represents a summary of a backup run
type BackupRunSummary struct {
	ID              uuid.UUID `json:"id"`
	AgentID         uuid.UUID `json:"agent_id"`
	PolicyID        uuid.UUID `json:"policy_id"`
	StartTime       string    `json:"start_time"`
	EndTime         *string   `json:"end_time,omitempty"`
	Status          string    `json:"status"`
	DurationSeconds *float64  `json:"duration_seconds,omitempty"`
	SnapshotID      *string   `json:"snapshot_id,omitempty"`
	ErrorMessage    *string   `json:"error_message,omitempty"`
}

// BackupRunDetailResponse represents detailed backup run information with logs
type BackupRunDetailResponse struct {
	ID              uuid.UUID `json:"id"`
	AgentID         uuid.UUID `json:"agent_id"`
	PolicyID        uuid.UUID `json:"policy_id"`
	StartTime       string    `json:"start_time"`
	EndTime         *string   `json:"end_time,omitempty"`
	Status          string    `json:"status"`
	DurationSeconds *float64  `json:"duration_seconds,omitempty"`
	SnapshotID      *string   `json:"snapshot_id,omitempty"`
	ErrorMessage    *string   `json:"error_message,omitempty"`
	Log             string    `json:"log,omitempty"`
}

// handleGetBackupRuns retrieves backup runs for an agent with filtering and pagination
func (a *API) handleGetBackupRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID from URL path
	// URL format: /agents/{id}/backup-runs
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "agents" || pathParts[2] != "backup-runs" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/backup-runs")
		return
	}

	agentIDStr := pathParts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
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

	// Parse query parameters
	statusFilter := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	limit := 100 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Build query
	query := a.store.GetDB().WithContext(ctx).
		Where("agent_id = ? AND tenant_id = ?", agentID, a.store.GetTenantID())

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	// Get total count
	var total int64
	query.Model(&store.BackupRun{}).Count(&total)

	// Get runs with limit
	var runs []store.BackupRun
	err = query.Order("start_time desc").Limit(limit).Find(&runs).Error
	if err != nil {
		log.Printf("Failed to retrieve backup runs for agent %s: %v", agentID, err)
		sendError(w, http.StatusInternalServerError, "Failed to retrieve backup runs", err.Error())
		return
	}

	// Convert to response format
	summaries := make([]BackupRunSummary, len(runs))
	for i, run := range runs {
		summary := BackupRunSummary{
			ID:              run.ID,
			AgentID:         run.AgentID,
			PolicyID:        run.PolicyID,
			StartTime:       run.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			Status:          run.Status,
			DurationSeconds: run.DurationSeconds,
			SnapshotID:      run.SnapshotID,
			ErrorMessage:    run.ErrorMessage,
		}
		if run.EndTime != nil {
			endTimeStr := run.EndTime.Format("2006-01-02T15:04:05Z07:00")
			summary.EndTime = &endTimeStr
		}
		summaries[i] = summary
	}

	response := BackupRunsResponse{
		Runs:  summaries,
		Total: int(total),
		Limit: limit,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetBackupRun retrieves a single backup run with logs
func (a *API) handleGetBackupRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID and run ID from URL path
	// URL format: /agents/{agentId}/backup-runs/{runId}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 4 || pathParts[0] != "agents" || pathParts[2] != "backup-runs" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{agentId}/backup-runs/{runId}")
		return
	}

	agentIDStr := pathParts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
		return
	}

	runIDStr := pathParts[3]
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid run ID", "Run ID must be a valid UUID")
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

	// Get backup run
	run, err := a.store.GetBackupRun(ctx, runID)
	if err != nil {
		log.Printf("Backup run not found: %s", runID)
		sendError(w, http.StatusNotFound, "Backup run not found", fmt.Sprintf("No backup run found with ID: %s", runID))
		return
	}

	// Verify run belongs to this agent
	if run.AgentID != agentID {
		sendError(w, http.StatusNotFound, "Backup run not found", "Backup run does not belong to this agent")
		return
	}

	// Get logs and reconstruct
	logs, err := a.store.GetBackupRunLogs(ctx, runID)
	if err != nil {
		log.Printf("Failed to retrieve logs for backup run %s: %v", runID, err)
		// Continue without logs rather than failing
	}

	var logContent strings.Builder
	for _, logEntry := range logs {
		logContent.WriteString(logEntry.Message)
	}

	// Build response
	response := BackupRunDetailResponse{
		ID:              run.ID,
		AgentID:         run.AgentID,
		PolicyID:        run.PolicyID,
		StartTime:       run.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		Status:          run.Status,
		DurationSeconds: run.DurationSeconds,
		SnapshotID:      run.SnapshotID,
		ErrorMessage:    run.ErrorMessage,
		Log:             logContent.String(),
	}

	if run.EndTime != nil {
		endTimeStr := run.EndTime.Format("2006-01-02T15:04:05Z07:00")
		response.EndTime = &endTimeStr
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
