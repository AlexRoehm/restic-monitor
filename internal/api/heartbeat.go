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

// DiskInfo represents disk usage information for a single mount point
type DiskInfo struct {
	MountPath  string `json:"mountPath"`
	FreeBytes  int64  `json:"freeBytes"`
	TotalBytes int64  `json:"totalBytes"`
}

// TaskTypeCount represents the count of running tasks by type
type TaskTypeCount struct {
	TaskType string `json:"taskType"`
	Count    int    `json:"count"`
}

// TaskTypeCapacity represents available capacity by task type
type TaskTypeCapacity struct {
	TaskType  string `json:"taskType"`
	Available int    `json:"available"`
	Maximum   int    `json:"maximum"`
}

// AgentHeartbeatRequest represents the heartbeat payload from an agent
type AgentHeartbeatRequest struct {
	Version          string     `json:"version"`
	OS               string     `json:"os"`
	UptimeSeconds    *int64     `json:"uptimeSeconds,omitempty"`
	Disks            []DiskInfo `json:"disks,omitempty"`
	LastBackupStatus string     `json:"lastBackupStatus,omitempty"` // success, failure, none, running
	// Load information (EPIC 15 Phase 3)
	CurrentTasksCount    *int                `json:"currentTasksCount,omitempty"`
	RunningTaskTypes     []TaskTypeCount     `json:"runningTaskTypes,omitempty"`
	AvailableSlots       *int                `json:"availableSlots,omitempty"`
	AvailableSlotsByType []TaskTypeCapacity  `json:"availableSlotsByType,omitempty"`
}

// AgentHeartbeatResponse represents the response to a heartbeat
type AgentHeartbeatResponse struct {
	Status                    string `json:"status"`
	NextTaskCheckAfterSeconds int    `json:"nextTaskCheckAfterSeconds"`
}

// handleAgentHeartbeat godoc
// @Summary Receive agent heartbeat
// @Description Processes periodic health and status updates from backup agents
// @Tags Agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID (UUID)"
// @Param request body AgentHeartbeatRequest true "Heartbeat data"
// @Success 200 {object} AgentHeartbeatResponse "Heartbeat processed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or validation error"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {object} ErrorResponse "Agent not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /agents/{id}/heartbeat [post]
func (a *API) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID from URL path
	// URL format: /agents/{id}/heartbeat
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "agents" || pathParts[2] != "heartbeat" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/heartbeat")
		return
	}

	agentIDStr := pathParts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
		return
	}

	// Parse request body
	var req AgentHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	validationErrors := validateHeartbeatRequest(req)
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

	// Find the agent
	var agent store.Agent
	err = a.store.GetDB().Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).First(&agent).Error
	if err != nil {
		log.Printf("Agent not found: %s (tenant: %s)", agentID, a.store.GetTenantID())
		sendError(w, http.StatusNotFound, "Agent not found", fmt.Sprintf("No agent found with ID: %s", agentID))
		return
	}

	// Track previous status for transition logging
	previousStatus := agent.Status

	// Update agent fields
	now := time.Now()
	agent.Version = req.Version
	agent.OS = req.OS
	agent.Status = "online"
	agent.LastSeenAt = &now

	if req.UptimeSeconds != nil {
		agent.UptimeSeconds = req.UptimeSeconds
	}

	if req.LastBackupStatus != "" {
		agent.LastBackupStatus = req.LastBackupStatus
	}

	if len(req.Disks) > 0 {
		// Convert disks to JSONB format
		disksJSON := make([]map[string]interface{}, len(req.Disks))
		for i, disk := range req.Disks {
			disksJSON[i] = map[string]interface{}{
				"mountPath":  disk.MountPath,
				"freeBytes":  disk.FreeBytes,
				"totalBytes": disk.TotalBytes,
			}
		}
		agent.FreeDisk = store.JSONB{
			"disks": disksJSON,
		}
	}

	// Save updated agent
	if err := a.store.GetDB().WithContext(ctx).Save(&agent).Error; err != nil {
		log.Printf("Failed to update agent %s: %v", agentID, err)
		sendError(w, http.StatusInternalServerError, "Failed to update agent", err.Error())
		return
	}

	// Log status transitions
	if previousStatus != "online" && agent.Status == "online" {
		log.Printf("Agent %s (%s) transitioned from %s to online", agent.ID, agent.Hostname, previousStatus)
	}

	// Log heartbeat details at debug level (can be made conditional based on log level)
	diskCount := 0
	if len(req.Disks) > 0 {
		diskCount = len(req.Disks)
	}
	log.Printf("Heartbeat from %s (%s): version=%s, os=%s, uptime=%v, disks=%d, backup_status=%s",
		agent.ID, agent.Hostname, agent.Version, agent.OS, req.UptimeSeconds, diskCount, req.LastBackupStatus)

	// Update agent backoff state (EPIC 15 Phase 6)
	if err := a.UpdateAgentBackoffState(agentID); err != nil {
		log.Printf("Failed to update backoff state for agent %s: %v", agentID, err)
		// Don't fail the heartbeat - backoff state is informational
	}

	// Return success response
	response := AgentHeartbeatResponse{
		Status:                    "ok",
		NextTaskCheckAfterSeconds: 30, // Default polling interval
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// validateHeartbeatRequest validates the heartbeat request
func validateHeartbeatRequest(req AgentHeartbeatRequest) []string {
	var errors []string

	if req.Version == "" {
		errors = append(errors, "version is required and cannot be empty")
	} else if len(req.Version) > 50 {
		errors = append(errors, "version must not exceed 50 characters")
	}

	if req.OS == "" {
		errors = append(errors, "os is required and cannot be empty")
	} else if len(req.OS) > 50 {
		errors = append(errors, "os must not exceed 50 characters")
	}

	if req.UptimeSeconds != nil && *req.UptimeSeconds < 0 {
		errors = append(errors, "uptimeSeconds must be >= 0")
	}

	if req.LastBackupStatus != "" {
		validStatuses := map[string]bool{
			"success": true,
			"failure": true,
			"none":    true,
			"running": true,
		}
		if !validStatuses[req.LastBackupStatus] {
			errors = append(errors, "lastBackupStatus must be one of: success, failure, none, running")
		}
	}

	// Validate disks
	for i, disk := range req.Disks {
		if disk.MountPath == "" {
			errors = append(errors, fmt.Sprintf("disk[%d]: mountPath is required", i))
		} else if len(disk.MountPath) > 255 {
			errors = append(errors, fmt.Sprintf("disk[%d]: mountPath must not exceed 255 characters", i))
		}

		if disk.FreeBytes < 0 {
			errors = append(errors, fmt.Sprintf("disk[%d]: freeBytes must be >= 0", i))
		}

		if disk.TotalBytes <= 0 {
			errors = append(errors, fmt.Sprintf("disk[%d]: totalBytes must be > 0", i))
		}
	}

	if len(req.Disks) > 100 {
		errors = append(errors, "disks array must not exceed 100 items")
	}

	return errors
}
