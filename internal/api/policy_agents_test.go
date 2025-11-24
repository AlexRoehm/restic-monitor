package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/api"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestListPolicyAgentsHappyPath tests listing agents for a policy (TDD - Epic 7.6)
func TestListPolicyAgentsHappyPath(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	// Create policy
	policy := createTestPolicy(t, db, tenantID, "shared-policy")

	// Create three agents and assign policy to all of them
	now := time.Now()
	agent1 := store.Agent{
		TenantID:   tenantID,
		Hostname:   "web-server-01",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "online",
		LastSeenAt: &now,
	}
	db.Create(&agent1)

	fiveMinutesAgo := now.Add(-5 * time.Minute)
	agent2 := store.Agent{
		TenantID:   tenantID,
		Hostname:   "web-server-02",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "online",
		LastSeenAt: &fiveMinutesAgo,
	}
	db.Create(&agent2)

	tenMinutesAgo := now.Add(-10 * time.Minute)
	agent3 := store.Agent{
		TenantID:   tenantID,
		Hostname:   "app-server-01",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "offline",
		LastSeenAt: &tenMinutesAgo,
	}
	db.Create(&agent3)

	// Create assignments
	for _, agent := range []store.Agent{agent1, agent2, agent3} {
		link := store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy.ID,
		}
		db.Create(&link)
	}

	// Request agents for policy
	url := "/policies/" + policy.ID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, policy.ID.String(), response["policyId"])
	assert.Equal(t, policy.Name, response["policyName"])

	agents := response["agents"].([]interface{})
	assert.Len(t, agents, 3, "Should return all 3 agents")

	// Verify agent summaries are sorted by hostname
	firstAgent := agents[0].(map[string]interface{})
	assert.Equal(t, "app-server-01", firstAgent["hostname"])
	assert.Equal(t, "offline", firstAgent["status"])
}

// TestListPolicyAgentsNoPolicyAssignments tests policy with no agents (TDD - Epic 7.6)
func TestListPolicyAgentsNoPolicyAssignments(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	// Create policy with no assignments
	policy := createTestPolicy(t, db, tenantID, "unassigned-policy")

	url := "/policies/" + policy.ID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	agents := response["agents"].([]interface{})
	assert.Empty(t, agents, "Should return empty array for policy with no assignments")
}

// TestListPolicyAgentsPolicyNotFound tests listing agents for non-existent policy (TDD - Epic 7.6)
func TestListPolicyAgentsPolicyNotFound(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	nonExistentPolicyID := uuid.New()

	url := "/policies/" + nonExistentPolicyID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "policy not found")
}

// TestListPolicyAgentsInvalidPolicyUUID tests invalid UUID (TDD - Epic 7.6)
func TestListPolicyAgentsInvalidPolicyUUID(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	url := "/policies/invalid-uuid/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListPolicyAgentsCrossTenant tests cross-tenant policy access (TDD - Epic 7.6)
func TestListPolicyAgentsCrossTenant(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create policy in tenant1
	policy := createTestPolicy(t, db, tenant1, "tenant1-policy")

	// Request as tenant2
	url := "/policies/" + policy.ID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenant2.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Cross-tenant policy should appear as not found")
}

// TestListPolicyAgentsResponseSchema tests response structure (TDD - Epic 7.6)
func TestListPolicyAgentsResponseSchema(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	policy := createTestPolicy(t, db, tenantID, "test-policy")

	now := time.Now()
	agent := store.Agent{
		TenantID:   tenantID,
		Hostname:   "test-agent",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "online",
		LastSeenAt: &now,
	}
	db.Create(&agent)

	link := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	db.Create(&link)

	url := "/policies/" + policy.ID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify top-level fields
	assert.Contains(t, response, "policyId")
	assert.Contains(t, response, "policyName")
	assert.Contains(t, response, "agents")

	// Verify agent fields
	agents := response["agents"].([]interface{})
	require.Len(t, agents, 1)

	agentData := agents[0].(map[string]interface{})
	assert.Contains(t, agentData, "id")
	assert.Contains(t, agentData, "hostname")
	assert.Contains(t, agentData, "lastSeenAt")
	assert.Contains(t, agentData, "status")

	// Verify values
	assert.Equal(t, agent.Hostname, agentData["hostname"])
	assert.Equal(t, agent.Status, agentData["status"])
}

// TestListPolicyAgentsStatusCalculation tests online/offline status (TDD - Epic 7.6)
func TestListPolicyAgentsStatusCalculation(t *testing.T) {
	db := setupPolicyAgentsTestDB(t)
	tenantID := uuid.New()

	policy := createTestPolicy(t, db, tenantID, "test-policy")

	// Create online agent (seen 2 minutes ago)
	twoMinutesAgo := time.Now().Add(-2 * time.Minute)
	onlineAgent := store.Agent{
		TenantID:   tenantID,
		Hostname:   "online-agent",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "online",
		LastSeenAt: &twoMinutesAgo,
	}
	db.Create(&onlineAgent)

	// Create offline agent (seen 10 minutes ago)
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)
	offlineAgent := store.Agent{
		TenantID:   tenantID,
		Hostname:   "offline-agent",
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Status:     "offline",
		LastSeenAt: &tenMinutesAgo,
	}
	db.Create(&offlineAgent)

	// Assign both to policy
	for _, agent := range []store.Agent{onlineAgent, offlineAgent} {
		db.Create(&store.AgentPolicyLink{
			AgentID:  agent.ID,
			PolicyID: policy.ID,
		})
	}

	url := "/policies/" + policy.ID.String() + "/agents"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAgentsHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	agents := response["agents"].([]interface{})
	assert.Len(t, agents, 2)

	// Find each agent and verify status
	for _, agentData := range agents {
		agent := agentData.(map[string]interface{})
		hostname := agent["hostname"].(string)

		if hostname == "offline-agent" {
			assert.Equal(t, "offline", agent["status"], "Agent not seen in 5+ minutes should be offline")
		} else if hostname == "online-agent" {
			assert.Equal(t, "online", agent["status"], "Agent seen within 5 minutes should be online")
		}
	}
}

// Helper functions

func setupPolicyAgentsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	err = store.MigrateModels(db)
	require.NoError(t, err)

	return db
}
