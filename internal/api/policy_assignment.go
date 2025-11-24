package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PolicyAssignmentHandler handles policy assignment operations
type PolicyAssignmentHandler struct {
	db *gorm.DB
}

// NewPolicyAssignmentHandler creates a new policy assignment handler
func NewPolicyAssignmentHandler(db *gorm.DB) *PolicyAssignmentHandler {
	return &PolicyAssignmentHandler{db: db}
}

// ServeHTTP handles HTTP requests for policy assignments
func (h *PolicyAssignmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Extract agent ID and policy ID from URL path
	// Expected: /agents/{agentId}/policies/{policyId}
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(path, "/policies/")

	if len(parts) != 2 {
		writeJSONError(w, http.StatusBadRequest, "invalid URL format")
		return
	}

	agentIDStr := parts[0]
	policyIDStr := parts[1]

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid agent ID")
		return
	}

	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	// Route based on HTTP method
	switch r.Method {
	case http.MethodPost:
		h.handleAssignPolicy(w, r, tenantID, agentID, policyID)
	case http.MethodDelete:
		h.handleRemovePolicy(w, r, tenantID, agentID, policyID)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAssignPolicy handles POST /agents/{agentId}/policies/{policyId}
func (h *PolicyAssignmentHandler) handleAssignPolicy(w http.ResponseWriter, r *http.Request, tenantID, agentID, policyID uuid.UUID) {
	// Verify agent exists and belongs to tenant
	var agent store.Agent
	err := h.db.Where("id = ? AND tenant_id = ?", agentID, tenantID).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Policy assignment failed: agent %s not found (tenant: %s)", agentID, tenantID)
			writeJSONError(w, http.StatusNotFound, "agent not found")
			return
		}
		log.Printf("Error finding agent %s: %v", agentID, err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Verify policy exists and belongs to tenant
	var policy store.Policy
	err = h.db.Where("id = ? AND tenant_id = ?", policyID, tenantID).First(&policy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Policy assignment failed: policy %s not found (tenant: %s)", policyID, tenantID)
			writeJSONError(w, http.StatusNotFound, "policy not found")
			return
		}
		log.Printf("Error finding policy %s: %v", policyID, err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Check if assignment already exists
	var existingLink store.AgentPolicyLink
	err = h.db.Where("agent_id = ? AND policy_id = ?", agentID, policyID).First(&existingLink).Error
	if err == nil {
		// Assignment already exists
		log.Printf("Policy assignment failed: policy '%s' already assigned to agent '%s'", policy.Name, agent.Hostname)
		writeJSONError(w, http.StatusConflict, "policy already assigned to agent")
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error checking existing assignment: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Create assignment
	link := store.AgentPolicyLink{
		AgentID:  agentID,
		PolicyID: policyID,
	}
	err = h.db.Create(&link).Error
	if err != nil {
		log.Printf("Error creating policy assignment: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to assign policy")
		return
	}

	log.Printf("Assigned policy '%s' (ID: %s, schedule: %s) to agent '%s' (ID: %s, tenant: %s)",
		policy.Name, policyID, policy.Schedule, agent.Hostname, agentID, tenantID)

	// TODO: Increment Prometheus metric: policy_assign_total

	// Return success response
	response := map[string]interface{}{
		"status":   "assigned",
		"agentId":  agentID.String(),
		"policyId": policyID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleRemovePolicy handles DELETE /agents/{agentId}/policies/{policyId}
func (h *PolicyAssignmentHandler) handleRemovePolicy(w http.ResponseWriter, r *http.Request, tenantID, agentID, policyID uuid.UUID) {
	// Verify assignment exists
	var link store.AgentPolicyLink
	err := h.db.Where("agent_id = ? AND policy_id = ?", agentID, policyID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Policy removal failed: assignment not found (agent: %s, policy: %s, tenant: %s)",
				agentID, policyID, tenantID)
			writeJSONError(w, http.StatusNotFound, "assignment not found")
			return
		}
		log.Printf("Error finding assignment: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Delete assignment
	err = h.db.Delete(&link).Error
	if err != nil {
		log.Printf("Error deleting policy assignment (agent: %s, policy: %s): %v", agentID, policyID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to remove policy")
		return
	}

	log.Printf("Removed policy assignment (policy: %s, agent: %s, tenant: %s)", policyID, agentID, tenantID)

	// TODO: Increment Prometheus metric: policy_unassign_total

	// Return success response
	response := map[string]interface{}{
		"status": "removed",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
