package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// AgentSettingsUpdateRequest represents a request to update agent settings
type AgentSettingsUpdateRequest struct {
	MaxConcurrentTasks   *int `json:"max_concurrent_tasks,omitempty"`
	MaxConcurrentBackups *int `json:"max_concurrent_backups,omitempty"`
	MaxConcurrentChecks  *int `json:"max_concurrent_checks,omitempty"`
	MaxConcurrentPrunes  *int `json:"max_concurrent_prunes,omitempty"`
	CPUQuotaPercent      *int `json:"cpu_quota_percent,omitempty"`
	BandwidthLimitMbps   *int `json:"bandwidth_limit_mbps,omitempty"`
}

// AgentSettingsResponse represents agent settings in API responses
type AgentSettingsResponse struct {
	MaxConcurrentTasks   int  `json:"max_concurrent_tasks"`
	MaxConcurrentBackups int  `json:"max_concurrent_backups"`
	MaxConcurrentChecks  int  `json:"max_concurrent_checks"`
	MaxConcurrentPrunes  int  `json:"max_concurrent_prunes"`
	CPUQuotaPercent      int  `json:"cpu_quota_percent"`
	BandwidthLimitMbps   *int `json:"bandwidth_limit_mbps,omitempty"`
}

// handleUpdateAgentSettings handles PATCH /agents/{id}/settings
func (a *API) handleUpdateAgentSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Parse agent ID from URL
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 || parts[0] != "agents" || parts[2] != "settings" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/settings")
		return
	}

	agentIDStr := parts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
		return
	}

	// Parse request body
	var req AgentSettingsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate request
	if errs := validateAgentSettings(req); len(errs) > 0 {
		sendError(w, http.StatusBadRequest, "Validation failed", strings.Join(errs, "; "))
		return
	}

	// Fetch agent
	var agent store.Agent
	err = a.store.GetDB().WithContext(ctx).
		Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).
		First(&agent).Error

	if err != nil {
		log.Printf("Agent not found: %s", agentID)
		sendError(w, http.StatusNotFound, "Agent not found", fmt.Sprintf("No agent found with ID: %s", agentID))
		return
	}

	// Update settings
	updates := make(map[string]interface{})
	if req.MaxConcurrentTasks != nil {
		updates["max_concurrent_tasks"] = *req.MaxConcurrentTasks
	}
	if req.MaxConcurrentBackups != nil {
		updates["max_concurrent_backups"] = *req.MaxConcurrentBackups
	}
	if req.MaxConcurrentChecks != nil {
		updates["max_concurrent_checks"] = *req.MaxConcurrentChecks
	}
	if req.MaxConcurrentPrunes != nil {
		updates["max_concurrent_prunes"] = *req.MaxConcurrentPrunes
	}
	if req.CPUQuotaPercent != nil {
		updates["cpu_quota_percent"] = *req.CPUQuotaPercent
	}
	if req.BandwidthLimitMbps != nil {
		updates["bandwidth_limit_mbps"] = *req.BandwidthLimitMbps
	}

	if len(updates) == 0 {
		sendError(w, http.StatusBadRequest, "No updates provided", "At least one setting must be provided")
		return
	}

	// Apply updates
	err = a.store.GetDB().WithContext(ctx).
		Model(&agent).
		Updates(updates).Error

	if err != nil {
		log.Printf("Failed to update agent settings: %v", err)
		sendError(w, http.StatusInternalServerError, "Failed to update agent settings", err.Error())
		return
	}

	// Fetch updated agent
	err = a.store.GetDB().WithContext(ctx).
		Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).
		First(&agent).Error

	if err != nil {
		log.Printf("Failed to fetch updated agent: %v", err)
		sendError(w, http.StatusInternalServerError, "Failed to fetch updated agent", err.Error())
		return
	}

	// Build response
	response := AgentSettingsResponse{
		MaxConcurrentTasks:   getIntValue(agent.MaxConcurrentTasks, 1),
		MaxConcurrentBackups: getIntValue(agent.MaxConcurrentBackups, 1),
		MaxConcurrentChecks:  getIntValue(agent.MaxConcurrentChecks, 1),
		MaxConcurrentPrunes:  getIntValue(agent.MaxConcurrentPrunes, 1),
		CPUQuotaPercent:      getIntValue(agent.CPUQuotaPercent, 50),
		BandwidthLimitMbps:   agent.BandwidthLimitMbps,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// validateAgentSettings validates agent settings update request
func validateAgentSettings(req AgentSettingsUpdateRequest) []string {
	var errors []string

	if req.MaxConcurrentTasks != nil {
		if *req.MaxConcurrentTasks <= 0 {
			errors = append(errors, "max_concurrent_tasks must be positive")
		}
		if *req.MaxConcurrentTasks > 100 {
			errors = append(errors, "max_concurrent_tasks cannot exceed 100")
		}
	}

	if req.MaxConcurrentBackups != nil && *req.MaxConcurrentBackups < 0 {
		errors = append(errors, "max_concurrent_backups must be non-negative")
	}

	if req.MaxConcurrentChecks != nil && *req.MaxConcurrentChecks < 0 {
		errors = append(errors, "max_concurrent_checks must be non-negative")
	}

	if req.MaxConcurrentPrunes != nil && *req.MaxConcurrentPrunes < 0 {
		errors = append(errors, "max_concurrent_prunes must be non-negative")
	}

	if req.CPUQuotaPercent != nil {
		if *req.CPUQuotaPercent < 1 || *req.CPUQuotaPercent > 100 {
			errors = append(errors, "cpu_quota_percent must be between 1 and 100")
		}
	}

	if req.BandwidthLimitMbps != nil {
		if *req.BandwidthLimitMbps <= 0 {
			errors = append(errors, "bandwidth_limit_mbps must be positive if set")
		}
		if *req.BandwidthLimitMbps > 100000 {
			errors = append(errors, "bandwidth_limit_mbps cannot exceed 100000")
		}
	}

	return errors
}

// getIntValue returns the value of an int pointer or a default value if nil
func getIntValue(ptr *int, defaultVal int) int {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}
