package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/example/restic-monitor/internal/store"
)

// Agent registration request/response types

type AgentRegisterRequest struct {
	Hostname string                 `json:"hostname"`
	OS       string                 `json:"os"`
	Arch     string                 `json:"arch"`
	Version  string                 `json:"version"`
	IP       string                 `json:"ip,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AgentRegisterResponse struct {
	AgentID      string    `json:"agentId"`
	Hostname     string    `json:"hostname"`
	RegisteredAt time.Time `json:"registeredAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Message      string    `json:"message"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// handleAgentRegister godoc
// @Summary Register or update a backup agent
// @Description Registers a new backup agent or updates an existing agent's metadata. Uses hostname as unique identifier.
// @Tags Agents
// @Accept json
// @Produce json
// @Param request body AgentRegisterRequest true "Agent registration data"
// @Success 201 {object} AgentRegisterResponse "Agent registered successfully"
// @Success 200 {object} AgentRegisterResponse "Agent metadata updated"
// @Failure 400 {object} ErrorResponse "Invalid request body or validation error"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /agents/register [post]
func (a *API) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req AgentRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	validationErrors := validateAgentRequest(req)
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

	// Check if agent with this hostname already exists
	var existingAgent store.Agent
	err := a.store.GetDB().Where("hostname = ? AND tenant_id = ?", req.Hostname, a.store.GetTenantID()).First(&existingAgent).Error

	if err == nil {
		// Agent exists - update it
		existingAgent.OS = req.OS
		existingAgent.Arch = req.Arch
		existingAgent.Version = req.Version
		existingAgent.Status = "online"
		now := time.Now()
		existingAgent.LastSeenAt = &now

		if req.Metadata != nil {
			existingAgent.Metadata = store.JSONB(req.Metadata)
		}

		if err := a.store.GetDB().Save(&existingAgent).Error; err != nil {
			log.Printf("Failed to update agent %s: %v", req.Hostname, err)
			sendError(w, http.StatusInternalServerError, "Failed to update agent", err.Error())
			return
		}

		log.Printf("Updated agent: %s (ID: %s)", req.Hostname, existingAgent.ID)

		// Return updated agent
		response := AgentRegisterResponse{
			AgentID:      existingAgent.ID.String(),
			Hostname:     existingAgent.Hostname,
			RegisteredAt: existingAgent.CreatedAt,
			UpdatedAt:    existingAgent.UpdatedAt,
			Message:      "Agent metadata updated",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create new agent
	agent := store.Agent{
		Hostname: req.Hostname,
		OS:       req.OS,
		Arch:     req.Arch,
		Version:  req.Version,
		Status:   "online",
	}

	now := time.Now()
	agent.LastSeenAt = &now

	if req.Metadata != nil {
		agent.Metadata = store.JSONB(req.Metadata)
	}

	if err := a.store.CreateAgent(ctx, &agent); err != nil {
		log.Printf("Failed to create agent %s: %v", req.Hostname, err)
		sendError(w, http.StatusInternalServerError, "Failed to create agent", err.Error())
		return
	}

	log.Printf("Registered new agent: %s (ID: %s)", agent.Hostname, agent.ID)

	response := AgentRegisterResponse{
		AgentID:      agent.ID.String(),
		Hostname:     agent.Hostname,
		RegisteredAt: agent.CreatedAt,
		UpdatedAt:    agent.UpdatedAt,
		Message:      "Agent registered successfully",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func validateAgentRequest(req AgentRegisterRequest) []string {
	var errors []string

	if req.Hostname == "" {
		errors = append(errors, "hostname is required and cannot be empty")
	} else if len(req.Hostname) > 255 {
		errors = append(errors, "hostname must not exceed 255 characters")
	}

	if req.OS == "" {
		errors = append(errors, "os is required and cannot be empty")
	} else if len(req.OS) > 50 {
		errors = append(errors, "os must not exceed 50 characters")
	}

	if req.Arch == "" {
		errors = append(errors, "arch is required and cannot be empty")
	} else if len(req.Arch) > 50 {
		errors = append(errors, "arch must not exceed 50 characters")
	}

	if req.Version == "" {
		errors = append(errors, "version is required and cannot be empty")
	} else if len(req.Version) > 50 {
		errors = append(errors, "version must not exceed 50 characters")
	}

	return errors
}

func sendError(w http.ResponseWriter, status int, error string, details string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   error,
		Details: details,
	})
}
