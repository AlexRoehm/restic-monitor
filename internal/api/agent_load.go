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

// AgentLoadResponse represents the current load status of an agent
type AgentLoadResponse struct {
	AgentID              string             `json:\"agentId\"`
	Hostname             string             `json:\"hostname\"`
	Status               string             `json:\"status\"`
	CurrentTasksCount    int                `json:\"currentTasksCount\"`
	MaxConcurrentTasks   int                `json:\"maxConcurrentTasks\"`
	AvailableSlots       int                `json:\"availableSlots\"`
	RunningTaskTypes     []TaskTypeCount    `json:\"runningTaskTypes,omitempty\"`
	AvailableSlotsByType []TaskTypeCapacity `json:\"availableSlotsByType,omitempty\"`
	IsSaturated          bool               `json:\"isSaturated\"`
	LastSeenAt           *string            `json:\"lastSeenAt,omitempty\"`
}

// handleGetAgentLoad handles GET /agents/{id}/load
func (a *API) handleGetAgentLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Parse agent ID from URL
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 || parts[0] != "agents" || parts[2] != "load" {
		sendError(w, http.StatusBadRequest, "Invalid URL format", "Expected /agents/{id}/load")
		return
	}

	agentIDStr := parts[1]
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
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

	// Build response
	maxTasks := 1
	if agent.MaxConcurrentTasks != nil {
		maxTasks = *agent.MaxConcurrentTasks
	}

	// For now, we don't have real-time running task data stored in DB
	// This would come from the most recent heartbeat's load information
	// In a production system, you'd store this in a cache or separate table
	currentTasks := 0
	availableSlots := maxTasks

	response := AgentLoadResponse{
		AgentID:            agent.ID.String(),
		Hostname:           agent.Hostname,
		Status:             agent.Status,
		CurrentTasksCount:  currentTasks,
		MaxConcurrentTasks: maxTasks,
		AvailableSlots:     availableSlots,
		IsSaturated:        availableSlots == 0,
	}

	if agent.LastSeenAt != nil {
		lastSeenStr := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		response.LastSeenAt = &lastSeenStr
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
