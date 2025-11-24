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

// PolicyRequest represents the request body for creating/updating policies
type PolicyRequest struct {
	Name               string                 `json:"name"`
	Description        *string                `json:"description,omitempty"`
	Schedule           string                 `json:"schedule"`
	IncludePaths       []string               `json:"includePaths"`
	ExcludePaths       []string               `json:"excludePaths,omitempty"`
	Repository         map[string]interface{} `json:"repository"`
	RetentionRules     map[string]interface{} `json:"retentionRules"`
	BandwidthLimitKBps *int                   `json:"bandwidthLimitKBps,omitempty"`
	ParallelFiles      *int                   `json:"parallelFiles,omitempty"`
	Enabled            *bool                  `json:"enabled,omitempty"`
}

// PolicyResponse represents the response body for policy operations
type PolicyResponse struct {
	ID                 string                 `json:"id"`
	TenantID           string                 `json:"tenantId"`
	Name               string                 `json:"name"`
	Description        *string                `json:"description,omitempty"`
	Schedule           string                 `json:"schedule"`
	IncludePaths       []string               `json:"includePaths"`
	ExcludePaths       []string               `json:"excludePaths,omitempty"`
	Repository         map[string]interface{} `json:"repository"`
	RetentionRules     map[string]interface{} `json:"retentionRules"`
	BandwidthLimitKBps *int                   `json:"bandwidthLimitKBps,omitempty"`
	ParallelFiles      *int                   `json:"parallelFiles,omitempty"`
	Enabled            bool                   `json:"enabled"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// handlePolicies routes policy requests to the appropriate handler
func (a *API) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.handleCreatePolicy(w, r)
	case http.MethodGet:
		// Check if we're listing all policies or getting a single policy
		path := strings.TrimPrefix(r.URL.Path, "/policies")
		if path == "" || path == "/" {
			a.handleListPolicies(w, r)
		} else {
			a.handleGetPolicy(w, r)
		}
	case http.MethodPut:
		a.handleUpdatePolicy(w, r)
	case http.MethodDelete:
		a.handleDeletePolicy(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreatePolicy creates a new policy
func (a *API) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the complete policy request
	if err := validatePolicyRequest(&req); err != nil {
		log.Printf("Policy validation failed for '%s': %v", req.Name, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract repository type and URL
	repoType, ok := req.Repository["type"].(string)
	if !ok || repoType == "" {
		http.Error(w, "repository.type is required", http.StatusBadRequest)
		return
	}

	// Build repository URL from repository config
	repoURL := buildRepositoryURL(req.Repository)

	// Convert arrays and maps to JSONB
	includePathsJSON := store.JSONB{"paths": req.IncludePaths}
	var excludePathsJSON store.JSONB
	if len(req.ExcludePaths) > 0 {
		excludePathsJSON = store.JSONB{"patterns": req.ExcludePaths}
	}
	repositoryConfigJSON := store.JSONB(req.Repository)
	retentionRulesJSON := store.JSONB(req.RetentionRules)

	policy := store.Policy{
		TenantID:           a.store.GetTenantID(),
		Name:               req.Name,
		Description:        req.Description,
		Schedule:           req.Schedule,
		IncludePaths:       includePathsJSON,
		ExcludePaths:       excludePathsJSON,
		RepositoryURL:      repoURL,
		RepositoryType:     repoType,
		RepositoryConfig:   repositoryConfigJSON,
		RetentionRules:     retentionRulesJSON,
		BandwidthLimitKBps: req.BandwidthLimitKBps,
		ParallelFiles:      req.ParallelFiles,
		Enabled:            true, // Default to enabled
	}

	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	// Create policy in database
	if err := a.store.CreatePolicy(ctx, &policy); err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			http.Error(w, "Policy with this name already exists", http.StatusConflict)
			return
		}
		log.Printf("Failed to create policy: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create policy: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Created policy: %s (ID: %s, schedule: %s, repository: %s, enabled: %v)",
		policy.Name, policy.ID, policy.Schedule, policy.RepositoryType, policy.Enabled)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(policyToResponse(policy))
}

// handleListPolicies returns all policies
func (a *API) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	policies, err := a.store.ListPolicies(ctx)
	if err != nil {
		log.Printf("Failed to list policies: %v", err)
		http.Error(w, fmt.Sprintf("Failed to list policies: %v", err), http.StatusInternalServerError)
		return
	}

	responses := make([]PolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = policyToResponse(policy)
	}

	log.Printf("Listed %d policies for tenant %s", len(policies), a.store.GetTenantID())
	json.NewEncoder(w).Encode(responses)
}

// handleGetPolicy returns a single policy by ID
func (a *API) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract policy ID from path: /policies/{id}
	path := strings.TrimPrefix(r.URL.Path, "/policies/")
	policyID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	policy, err := a.store.GetPolicy(ctx, policyID)
	if err != nil {
		log.Printf("Policy not found: %s (tenant: %s)", policyID, a.store.GetTenantID())
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	log.Printf("Retrieved policy: %s (ID: %s)", policy.Name, policy.ID)
	json.NewEncoder(w).Encode(policyToResponse(policy))
}

// handleUpdatePolicy updates an existing policy
func (a *API) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract policy ID from path: /policies/{id}
	path := strings.TrimPrefix(r.URL.Path, "/policies/")
	policyID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	// Get existing policy
	policy, err := a.store.GetPolicy(ctx, policyID)
	if err != nil {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate individual fields if provided (partial updates allowed)
	if req.Name != "" {
		if err := validatePolicyName(req.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.Name = req.Name
	}
	if req.Description != nil {
		policy.Description = req.Description
	}
	if req.Schedule != "" {
		if err := validateCronSchedule(req.Schedule); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.Schedule = req.Schedule
	}
	if len(req.IncludePaths) > 0 {
		if err := validateIncludePaths(req.IncludePaths); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.IncludePaths = store.JSONB{"paths": req.IncludePaths}
	}
	if len(req.ExcludePaths) > 0 {
		if err := validateExcludePaths(req.ExcludePaths); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.ExcludePaths = store.JSONB{"patterns": req.ExcludePaths}
	}
	if len(req.Repository) > 0 {
		repoType, ok := req.Repository["type"].(string)
		if !ok || repoType == "" {
			http.Error(w, "repository.type is required", http.StatusBadRequest)
			return
		}

		// Validate repository type and config
		if err := validateRepositoryType(repoType); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch repoType {
		case "s3":
			if err := validateS3Repository(req.Repository); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "rest-server":
			if err := validateRestServerRepository(req.Repository); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "fs":
			if err := validateFilesystemRepository(req.Repository); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "sftp":
			if err := validateSFTPRepository(req.Repository); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		policy.RepositoryType = repoType
		policy.RepositoryURL = buildRepositoryURL(req.Repository)
		policy.RepositoryConfig = store.JSONB(req.Repository)
	}
	if len(req.RetentionRules) > 0 {
		if err := validateRetentionRules(req.RetentionRules); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.RetentionRules = store.JSONB(req.RetentionRules)
	}
	if req.BandwidthLimitKBps != nil {
		if err := validateBandwidthLimit(req.BandwidthLimitKBps); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.BandwidthLimitKBps = req.BandwidthLimitKBps
	}
	if req.ParallelFiles != nil {
		if err := validateParallelFiles(req.ParallelFiles); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		policy.ParallelFiles = req.ParallelFiles
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	// Save updated policy
	if err := a.store.GetDB().Save(&policy).Error; err != nil {
		log.Printf("Failed to update policy: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update policy: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Updated policy: %s (ID: %s, enabled: %v)", policy.Name, policy.ID, policy.Enabled)

	json.NewEncoder(w).Encode(policyToResponse(policy))
}

// handleDeletePolicy deletes a policy
func (a *API) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract policy ID from path: /policies/{id}
	path := strings.TrimPrefix(r.URL.Path, "/policies/")
	policyID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	// Check if policy exists
	policy, err := a.store.GetPolicy(ctx, policyID)
	if err != nil {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	// Check if policy has agent assignments
	var linkCount int64
	a.store.GetDB().Model(&store.AgentPolicyLink{}).Where("policy_id = ?", policyID).Count(&linkCount)
	if linkCount > 0 {
		http.Error(w, "Cannot delete policy with agent assignments", http.StatusConflict)
		return
	}

	// Delete policy
	if err := a.store.GetDB().Delete(&policy).Error; err != nil {
		log.Printf("Failed to delete policy: %v", err)
		http.Error(w, fmt.Sprintf("Failed to delete policy: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted policy: %s (ID: %s, tenant: %s)", policy.Name, policy.ID, a.store.GetTenantID())

	w.WriteHeader(http.StatusNoContent)
}

// policyToResponse converts a store.Policy to PolicyResponse
func policyToResponse(policy store.Policy) PolicyResponse {
	// Extract include paths from JSONB
	var includePaths []string
	if pathsData, ok := policy.IncludePaths["paths"]; ok {
		if pathsList, ok := pathsData.([]interface{}); ok {
			for _, p := range pathsList {
				if pathStr, ok := p.(string); ok {
					includePaths = append(includePaths, pathStr)
				}
			}
		}
	}

	// Extract exclude paths from JSONB
	var excludePaths []string
	if policy.ExcludePaths != nil {
		if patternsData, ok := policy.ExcludePaths["patterns"]; ok {
			if patternsList, ok := patternsData.([]interface{}); ok {
				for _, p := range patternsList {
					if patternStr, ok := p.(string); ok {
						excludePaths = append(excludePaths, patternStr)
					}
				}
			}
		}
	}

	return PolicyResponse{
		ID:                 policy.ID.String(),
		TenantID:           policy.TenantID.String(),
		Name:               policy.Name,
		Description:        policy.Description,
		Schedule:           policy.Schedule,
		IncludePaths:       includePaths,
		ExcludePaths:       excludePaths,
		Repository:         policy.RepositoryConfig,
		RetentionRules:     policy.RetentionRules,
		BandwidthLimitKBps: policy.BandwidthLimitKBps,
		ParallelFiles:      policy.ParallelFiles,
		Enabled:            policy.Enabled,
		CreatedAt:          policy.CreatedAt,
		UpdatedAt:          policy.UpdatedAt,
	}
}

// buildRepositoryURL constructs a repository URL from repository config
func buildRepositoryURL(repo map[string]interface{}) string {
	repoType, _ := repo["type"].(string)

	switch repoType {
	case "s3":
		bucket, _ := repo["bucket"].(string)
		prefix, _ := repo["prefix"].(string)
		if prefix != "" {
			return fmt.Sprintf("s3:s3.amazonaws.com/%s/%s", bucket, prefix)
		}
		return fmt.Sprintf("s3:s3.amazonaws.com/%s", bucket)
	case "rest-server", "rest":
		url, _ := repo["url"].(string)
		return url
	case "fs", "local":
		path, _ := repo["path"].(string)
		return path
	case "sftp":
		host, _ := repo["host"].(string)
		path, _ := repo["path"].(string)
		return fmt.Sprintf("sftp:%s:%s", host, path)
	default:
		return ""
	}
}
