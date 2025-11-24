package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/restic-monitor/internal/api"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAssignPolicyToAgentHappyPath tests successful policy assignment (TDD - Epic 7.3)
func TestAssignPolicyToAgentHappyPath(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create agent and policy
	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	// Create request
	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "assigned", response["status"])
	assert.Equal(t, agent.ID.String(), response["agentId"])
	assert.Equal(t, policy.ID.String(), response["policyId"])

	// Verify assignment exists in database
	var link store.AgentPolicyLink
	err = db.First(&link, "agent_id = ? AND policy_id = ?", agent.ID, policy.ID).Error
	assert.NoError(t, err, "Assignment should exist in database")
}

// TestAssignPolicyAgentNotFound tests assignment when agent doesn't exist (TDD - Epic 7.3)
func TestAssignPolicyAgentNotFound(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create only policy (no agent)
	policy := createTestPolicy(t, db, tenantID, "test-policy")
	nonExistentAgentID := uuid.New()

	// Create request
	url := "/agents/" + nonExistentAgentID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "agent not found")
}

// TestAssignPolicyPolicyNotFound tests assignment when policy doesn't exist (TDD - Epic 7.3)
func TestAssignPolicyPolicyNotFound(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create only agent (no policy)
	agent := createTestAgent(t, db, tenantID, "test-agent")
	nonExistentPolicyID := uuid.New()

	// Create request
	url := "/agents/" + agent.ID.String() + "/policies/" + nonExistentPolicyID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "policy not found")
}

// TestAssignPolicyDuplicateAssignment tests duplicate assignment prevention (TDD - Epic 7.3)
func TestAssignPolicyDuplicateAssignment(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create agent and policy
	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	// Create first assignment directly in DB
	link := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err := db.Create(&link).Error
	require.NoError(t, err)

	// Attempt duplicate assignment via API
	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "already assigned")
}

// TestAssignPolicyInvalidAgentUUID tests invalid agent UUID (TDD - Epic 7.3)
func TestAssignPolicyInvalidAgentUUID(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	policy := createTestPolicy(t, db, tenantID, "test-policy")

	// Create request with invalid agent UUID
	url := "/agents/invalid-uuid/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "invalid agent ID")
}

// TestAssignPolicyInvalidPolicyUUID tests invalid policy UUID (TDD - Epic 7.3)
func TestAssignPolicyInvalidPolicyUUID(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	agent := createTestAgent(t, db, tenantID, "test-agent")

	// Create request with invalid policy UUID
	url := "/agents/" + agent.ID.String() + "/policies/invalid-uuid"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "invalid policy ID")
}

// TestAssignPolicyUnauthorizedTenant tests cross-tenant assignment prevention (TDD - Epic 7.3)
func TestAssignPolicyUnauthorizedTenant(t *testing.T) {
	db := setupTestDB(t)
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create agent in tenant1, policy in tenant2
	agent := createTestAgent(t, db, tenant1, "agent-tenant1")
	policy := createTestPolicy(t, db, tenant2, "policy-tenant2")

	// Attempt assignment as tenant1
	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenant1.String())

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusNotFound, w.Code, "Cross-tenant policy should appear as not found")
}

// TestAssignPolicyMissingTenantID tests request without tenant ID (TDD - Epic 7.3)
func TestAssignPolicyMissingTenantID(t *testing.T) {
	db := setupTestDB(t)

	// Create request without tenant ID header
	url := "/agents/" + uuid.New().String() + "/policies/" + uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, url, nil)

	// Execute
	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "tenant ID required")
}

// TestAssignPolicyResponseSchema tests response JSON schema (TDD - Epic 7.3)
func TestAssignPolicyResponseSchema(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify required fields exist
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "agentId")
	assert.Contains(t, response, "policyId")

	// Verify field types
	assert.IsType(t, "", response["status"])
	assert.IsType(t, "", response["agentId"])
	assert.IsType(t, "", response["policyId"])

	// Verify agentId and policyId are valid UUIDs
	_, err = uuid.Parse(response["agentId"].(string))
	assert.NoError(t, err, "agentId should be valid UUID")

	_, err = uuid.Parse(response["policyId"].(string))
	assert.NoError(t, err, "policyId should be valid UUID")
}

// TestRemovePolicyFromAgentHappyPath tests successful policy removal (TDD - Epic 7.4)
func TestRemovePolicyFromAgentHappyPath(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create agent, policy, and assignment
	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	link := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err := db.Create(&link).Error
	require.NoError(t, err)

	// Remove assignment
	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "removed", response["status"])

	// Verify assignment removed from database
	var count int64
	db.Model(&store.AgentPolicyLink{}).Where("agent_id = ? AND policy_id = ?", agent.ID, policy.ID).Count(&count)
	assert.Equal(t, int64(0), count, "Assignment should be removed from database")
}

// TestRemovePolicyNonexistentAssignment tests removing non-existent assignment (TDD - Epic 7.4)
func TestRemovePolicyNonexistentAssignment(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	// Create agent and policy but NO assignment
	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	// Attempt to remove non-existent assignment
	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "assignment not found")
}

// TestRemovePolicyAgentNotFound tests removal when agent doesn't exist (TDD - Epic 7.4)
func TestRemovePolicyAgentNotFound(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	policy := createTestPolicy(t, db, tenantID, "test-policy")
	nonExistentAgentID := uuid.New()

	url := "/agents/" + nonExistentAgentID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	// Verify response - could be 404 for agent or assignment
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRemovePolicyInvalidUUID tests removal with invalid UUID (TDD - Epic 7.4)
func TestRemovePolicyInvalidUUID(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	url := "/agents/invalid-uuid/policies/" + uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRemovePolicyResponseSchema tests DELETE response schema (TDD - Epic 7.4)
func TestRemovePolicyResponseSchema(t *testing.T) {
	db := setupTestDB(t)
	tenantID := uuid.New()

	agent := createTestAgent(t, db, tenantID, "test-agent")
	policy := createTestPolicy(t, db, tenantID, "test-policy")

	link := store.AgentPolicyLink{
		AgentID:  agent.ID,
		PolicyID: policy.ID,
	}
	err := db.Create(&link).Error
	require.NoError(t, err)

	url := "/agents/" + agent.ID.String() + "/policies/" + policy.ID.String()
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	handler := api.NewPolicyAssignmentHandler(db)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify required field exists
	assert.Contains(t, response, "status")
	assert.Equal(t, "removed", response["status"])
}

// Helper functions

func setupTestDB(t *testing.T) *gorm.DB {
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

func createTestAgent(t *testing.T, db *gorm.DB, tenantID uuid.UUID, hostname string) store.Agent {
	agent := store.Agent{
		TenantID: tenantID,
		Hostname: hostname,
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err := db.Create(&agent).Error
	require.NoError(t, err)
	return agent
}

func createTestPolicy(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name string) store.Policy {
	policy := store.Policy{
		TenantID:       tenantID,
		Name:           name,
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/data"}},
		RepositoryURL:  "s3:bucket/path",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}
	err := db.Create(&policy).Error
	require.NoError(t, err)
	return policy
}
