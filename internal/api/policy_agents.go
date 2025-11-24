package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PolicyAgentsHandler handles listing agents assigned to a policy
type PolicyAgentsHandler struct {
	db *gorm.DB
}

// NewPolicyAgentsHandler creates a new policy agents handler
func NewPolicyAgentsHandler(db *gorm.DB) *PolicyAgentsHandler {
	return &PolicyAgentsHandler{db: db}
}

// AgentSummary represents a summary of an agent for policy listing
type AgentSummary struct {
	ID         string     `json:"id"`
	Hostname   string     `json:"hostname"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
	Status     string     `json:"status"`
}

// ServeHTTP handles HTTP requests for policy agents listing
func (h *PolicyAgentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from header
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusUnauthorized, "tenant ID required")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	// Extract policy ID from URL path
	// Expected: /policies/{policyId}/agents
	path := strings.TrimPrefix(r.URL.Path, "/policies/")
	path = strings.TrimSuffix(path, "/agents")
	policyIDStr := strings.Trim(path, "/")

	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	// Only allow GET
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.handleListPolicyAgents(w, r, tenantID, policyID)
}

// handleListPolicyAgents handles GET /policies/{policyId}/agents
func (h *PolicyAgentsHandler) handleListPolicyAgents(w http.ResponseWriter, r *http.Request, tenantID, policyID uuid.UUID) {
	// Verify policy exists and belongs to tenant
	var policy store.Policy
	err := h.db.Where("id = ? AND tenant_id = ?", policyID, tenantID).First(&policy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSONError(w, http.StatusNotFound, "policy not found")
			return
		}
		log.Printf("Error finding policy: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Get agent IDs assigned to this policy
	var links []store.AgentPolicyLink
	err = h.db.Where("policy_id = ?", policyID).Find(&links).Error
	if err != nil {
		log.Printf("Error finding policy assignments: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Extract agent IDs
	agentIDs := make([]uuid.UUID, len(links))
	for i, link := range links {
		agentIDs[i] = link.AgentID
	}

	// Get all assigned agents
	var agents []store.Agent
	if len(agentIDs) > 0 {
		err = h.db.Where("id IN ? AND tenant_id = ?", agentIDs, tenantID).
			Order("hostname ASC").
			Find(&agents).Error
		if err != nil {
			log.Printf("Error finding agents: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	// Convert to agent summaries with status calculation
	summaries := make([]AgentSummary, 0, len(agents))
	now := time.Now()
	for _, agent := range agents {
		status := calculateAgentStatus(&agent, now)
		summary := AgentSummary{
			ID:         agent.ID.String(),
			Hostname:   agent.Hostname,
			LastSeenAt: agent.LastSeenAt,
			Status:     status,
		}
		summaries = append(summaries, summary)
	}

	log.Printf("Returned %d agents for policy '%s' (ID: %s)", len(summaries), policy.Name, policyID)

	// Return response
	response := map[string]interface{}{
		"policyId":   policyID.String(),
		"policyName": policy.Name,
		"agents":     summaries,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// calculateAgentStatus determines if an agent is online or offline based on last seen time
func calculateAgentStatus(agent *store.Agent, now time.Time) string {
	if agent.LastSeenAt == nil {
		return "offline"
	}

	// Online if seen within last 5 minutes
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	if agent.LastSeenAt.After(fiveMinutesAgo) {
		return "online"
	}

	return "offline"
}
