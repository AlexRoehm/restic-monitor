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

// handleAgentsRouter routes /agents/* requests to appropriate handlers
func (a *API) handleAgentsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// /agents/{id}/heartbeat -> heartbeat handler
	if len(parts) == 3 && parts[0] == "agents" && parts[2] == "heartbeat" {
		a.handleAgentHeartbeat(w, r)
		return
	}

	// /agents/{id}/policies -> policies handler
	if len(parts) == 3 && parts[0] == "agents" && parts[2] == "policies" && r.Method == http.MethodGet {
		a.handleGetAgentPolicies(w, r)
		return
	}

	// /agents/{id}/tasks -> tasks handler (EPIC 10.2)
	if len(parts) == 3 && parts[0] == "agents" && parts[2] == "tasks" && r.Method == http.MethodGet {
		a.handleGetAgentTasks(w, r)
		return
	}

	// /agents/{id}/tasks/{taskId}/ack -> task acknowledgment handler (EPIC 10.4)
	if len(parts) == 5 && parts[0] == "agents" && parts[2] == "tasks" && parts[4] == "ack" && r.Method == http.MethodPost {
		a.handleAcknowledgeTask(w, r)
		return
	}

	// /agents/{id}/task-results -> task result submission handler (EPIC 13.2)
	if len(parts) == 3 && parts[0] == "agents" && parts[2] == "task-results" && r.Method == http.MethodPost {
		a.handleTaskResults(w, r)
		return
	}

	// /agents/{id}/backup-runs/{runId} -> get single backup run with logs (EPIC 13.6)
	if len(parts) == 4 && parts[0] == "agents" && parts[2] == "backup-runs" && r.Method == http.MethodGet {
		a.handleGetBackupRun(w, r)
		return
	}

	// /agents/{id}/backup-runs -> get backup runs list (EPIC 13.6)
	if len(parts) == 3 && parts[0] == "agents" && parts[2] == "backup-runs" && r.Method == http.MethodGet {
		a.handleGetBackupRuns(w, r)
		return
	}

	// /agents/{id} -> GET handler
	if len(parts) == 2 && parts[0] == "agents" && r.Method == http.MethodGet {
		a.handleGetAgents(w, r)
		return
	}

	// Unknown pattern
	http.NotFound(w, r)
}

// AgentResponse represents a single agent in API responses
type AgentResponse struct {
	ID               string                 `json:"id"`
	Hostname         string                 `json:"hostname"`
	OS               string                 `json:"os"`
	Arch             string                 `json:"arch"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"`
	LastSeenAt       *string                `json:"last_seen_at,omitempty"`
	LastBackupStatus string                 `json:"last_backup_status,omitempty"`
	UptimeSeconds    *int64                 `json:"uptime_seconds,omitempty"`
	FreeDisk         map[string]interface{} `json:"free_disk,omitempty"`
	TotalFreeBytes   int64                  `json:"total_free_bytes,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

// AgentsListResponse represents the list of agents
type AgentsListResponse struct {
	Agents []AgentResponse `json:"agents"`
	Total  int             `json:"total"`
}

// handleGetAgents handles both GET /agents (list) and GET /agents/{id} (single)
func (a *API) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Parse URL to determine if it's list or single agent
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// GET /agents - list all agents
	if len(parts) == 1 && parts[0] == "agents" {
		var agents []store.Agent
		err := a.store.GetDB().WithContext(ctx).
			Where("tenant_id = ?", a.store.GetTenantID()).
			Order("hostname ASC").
			Find(&agents).Error

		if err != nil {
			log.Printf("Failed to fetch agents: %v", err)
			sendError(w, http.StatusInternalServerError, "Failed to fetch agents", err.Error())
			return
		}

		response := AgentsListResponse{
			Agents: make([]AgentResponse, len(agents)),
			Total:  len(agents),
		}

		for i, agent := range agents {
			response.Agents[i] = convertAgentToResponse(agent)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// GET /agents/{id} - get single agent
	if len(parts) == 2 && parts[0] == "agents" {
		agentIDStr := parts[1]
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid agent ID", "Agent ID must be a valid UUID")
			return
		}

		var agent store.Agent
		err = a.store.GetDB().WithContext(ctx).
			Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).
			First(&agent).Error

		if err != nil {
			log.Printf("Agent not found: %s", agentID)
			sendError(w, http.StatusNotFound, "Agent not found", fmt.Sprintf("No agent found with ID: %s", agentID))
			return
		}

		response := convertAgentToResponse(agent)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Invalid path
	http.NotFound(w, r)
}

// convertAgentToResponse converts a store.Agent to AgentResponse with calculated fields
func convertAgentToResponse(agent store.Agent) AgentResponse {
	response := AgentResponse{
		ID:               agent.ID.String(),
		Hostname:         agent.Hostname,
		OS:               agent.OS,
		Arch:             agent.Arch,
		Version:          agent.Version,
		Status:           agent.Status,
		LastBackupStatus: agent.LastBackupStatus,
		UptimeSeconds:    agent.UptimeSeconds,
		CreatedAt:        agent.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        agent.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if agent.LastSeenAt != nil {
		lastSeenStr := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		response.LastSeenAt = &lastSeenStr
	}

	if agent.FreeDisk != nil {
		response.FreeDisk = agent.FreeDisk

		// Calculate total free bytes
		if disksInterface, ok := agent.FreeDisk["disks"]; ok {
			if disks, ok := disksInterface.([]interface{}); ok {
				var totalFree int64
				for _, diskInterface := range disks {
					if disk, ok := diskInterface.(map[string]interface{}); ok {
						if freeBytes, ok := disk["freeBytes"]; ok {
							// Handle both float64 (from JSON) and int64
							switch v := freeBytes.(type) {
							case float64:
								totalFree += int64(v)
							case int64:
								totalFree += v
							case int:
								totalFree += int64(v)
							}
						}
					}
				}
				response.TotalFreeBytes = totalFree
			}
		}
	}

	if agent.Metadata != nil {
		response.Metadata = agent.Metadata
	}

	return response
}
