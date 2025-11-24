package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
)

// AgentPolicyResponse represents a policy in agent-friendly format
type AgentPolicyResponse struct {
	Name               string                 `json:"name"`
	Description        *string                `json:"description,omitempty"`
	Schedule           string                 `json:"schedule"`
	IncludePaths       []string               `json:"includePaths"`
	ExcludePaths       []string               `json:"excludePaths,omitempty"`
	Repository         map[string]interface{} `json:"repository"`
	RetentionRules     map[string]interface{} `json:"retentionRules"`
	BandwidthLimitKBps *int                   `json:"bandwidthLimitKBps,omitempty"`
	ParallelFiles      *int                   `json:"parallelFiles,omitempty"`
}

// handleGetAgentPolicies returns all enabled policies assigned to an agent
func (a *API) handleGetAgentPolicies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract agent ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	path = strings.TrimSuffix(path, "/policies")
	agentIDStr := strings.Trim(path, "/")

	// Parse agent ID
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		http.Error(w, "Invalid agent ID format", http.StatusBadRequest)
		return
	}

	// Verify agent exists and belongs to tenant
	var agent store.Agent
	if err := a.store.GetDB().Where("id = ? AND tenant_id = ?", agentID, a.store.GetTenantID()).First(&agent).Error; err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Get policy IDs assigned to this agent
	var links []store.AgentPolicyLink
	if err := a.store.GetDB().Where("agent_id = ?", agentID).Find(&links).Error; err != nil {
		log.Printf("Failed to get policy links for agent %s: %v", agentID, err)
		http.Error(w, "Failed to retrieve policies", http.StatusInternalServerError)
		return
	}

	// Extract policy IDs
	policyIDs := make([]uuid.UUID, len(links))
	for i, link := range links {
		policyIDs[i] = link.PolicyID
	}

	// Get all assigned policies (only enabled ones)
	var policies []store.Policy
	if len(policyIDs) > 0 {
		if err := a.store.GetDB().Where("id IN ? AND tenant_id = ? AND enabled = ?", policyIDs, a.store.GetTenantID(), true).Order("name ASC").Find(&policies).Error; err != nil {
			log.Printf("Failed to get policies for agent %s: %v", agentID, err)
			http.Error(w, "Failed to retrieve policies", http.StatusInternalServerError)
			return
		}
	}

	// Convert to agent-friendly format
	response := make([]AgentPolicyResponse, 0, len(policies))
	for _, policy := range policies {
		agentPolicy := agentPolicyToResponse(policy)
		response = append(response, agentPolicy)
	}

	// Return policies
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policies": response,
	})

	log.Printf("Returned %d policies for agent %s (%s)", len(response), agentID, agent.Hostname)
}

// agentPolicyToResponse converts a Policy to agent-friendly format
func agentPolicyToResponse(p store.Policy) AgentPolicyResponse {
	// Extract include paths from JSONB
	includePaths := []string{}
	if paths, ok := p.IncludePaths["paths"].([]interface{}); ok {
		for _, path := range paths {
			if str, ok := path.(string); ok {
				includePaths = append(includePaths, str)
			}
		}
	}

	// Extract exclude paths from JSONB
	var excludePaths []string
	if len(p.ExcludePaths) > 0 {
		if patterns, ok := p.ExcludePaths["patterns"].([]interface{}); ok {
			excludePaths = make([]string, 0, len(patterns))
			for _, pattern := range patterns {
				if str, ok := pattern.(string); ok {
					excludePaths = append(excludePaths, str)
				}
			}
		}
	}

	// Repository config is already in the right format
	repository := make(map[string]interface{})
	for k, v := range p.RepositoryConfig {
		repository[k] = v
	}

	// Retention rules are already in the right format
	retentionRules := make(map[string]interface{})
	for k, v := range p.RetentionRules {
		retentionRules[k] = v
	}

	return AgentPolicyResponse{
		Name:               p.Name,
		Description:        p.Description,
		Schedule:           p.Schedule,
		IncludePaths:       includePaths,
		ExcludePaths:       excludePaths,
		Repository:         repository,
		RetentionRules:     retentionRules,
		BandwidthLimitKBps: p.BandwidthLimitKBps,
		ParallelFiles:      p.ParallelFiles,
	}
}
