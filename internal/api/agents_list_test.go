package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetAgentsListRouterMapping tests that GET /agents endpoint exists (TDD)
func TestGetAgentsListRouterMapping(t *testing.T) {
	api, _ := setupTestAPI(t)

	req := httptest.NewRequest("GET", "/agents", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Route GET /agents should exist")
}

// TestGetAgentByIDRouterMapping tests that GET /agents/{id} endpoint exists (TDD)
func TestGetAgentByIDRouterMapping(t *testing.T) {
	api, st := setupTestAPI(t)
	agent := createTestAgent(t, st, "get-agent-by-id-test")

	req := httptest.NewRequest("GET", fmt.Sprintf("/agents/%s", agent.ID), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Route GET /agents/{id} should exist")
}

// TestGetAgentsListResponseSchema tests the list response format (TDD)
func TestGetAgentsListResponseSchema(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create multiple test agents
	agent1 := createTestAgent(t, st, "list-test-agent-1")
	agent2 := createTestAgent(t, st, "list-test-agent-2")
	agent3 := createTestAgent(t, st, "list-test-agent-3")

	// Update agent2 with more data
	now := time.Now()
	agent2.Status = "online"
	agent2.LastSeenAt = &now
	agent2.LastBackupStatus = "success"
	uptime := int64(86400)
	agent2.UptimeSeconds = &uptime
	agent2.FreeDisk = store.JSONB{
		"disks": []map[string]interface{}{
			{"mountPath": "/", "freeBytes": 50000000000, "totalBytes": 100000000000},
		},
	}
	st.GetDB().Save(agent2)

	req := httptest.NewRequest("GET", "/agents", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response has agents array
	agents, ok := response["agents"].([]interface{})
	require.True(t, ok, "Response should contain 'agents' array")
	assert.GreaterOrEqual(t, len(agents), 3, "Should have at least 3 agents")

	// Verify first agent has all required fields
	if len(agents) > 0 {
		agent := agents[0].(map[string]interface{})

		// Required fields
		assert.Contains(t, agent, "id")
		assert.Contains(t, agent, "hostname")
		assert.Contains(t, agent, "os")
		assert.Contains(t, agent, "arch")
		assert.Contains(t, agent, "version")
		assert.Contains(t, agent, "status")
		assert.Contains(t, agent, "created_at")
		assert.Contains(t, agent, "updated_at")

		// Optional fields (may be present)
		// last_seen_at, last_backup_status, uptime_seconds, total_free_bytes
	}

	// Verify total count
	assert.Contains(t, response, "total")
	total, ok := response["total"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, total, float64(3))

	_ = agent1
	_ = agent3
}

// TestGetAgentByIDResponseSchema tests single agent response format (TDD)
func TestGetAgentByIDResponseSchema(t *testing.T) {
	api, st := setupTestAPI(t)

	agent := createTestAgent(t, st, "single-agent-test")

	// Add full data
	now := time.Now()
	agent.Status = "online"
	agent.LastSeenAt = &now
	agent.LastBackupStatus = "success"
	uptime := int64(432000)
	agent.UptimeSeconds = &uptime
	agent.FreeDisk = store.JSONB{
		"disks": []map[string]interface{}{
			{"mountPath": "/", "freeBytes": 50000000000, "totalBytes": 100000000000},
			{"mountPath": "/data", "freeBytes": 200000000000, "totalBytes": 500000000000},
		},
	}
	st.GetDB().Save(agent)

	req := httptest.NewRequest("GET", fmt.Sprintf("/agents/%s", agent.ID), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, agent.ID.String(), response["id"])
	assert.Equal(t, agent.Hostname, response["hostname"])
	assert.Equal(t, agent.OS, response["os"])
	assert.Equal(t, agent.Arch, response["arch"])
	assert.Equal(t, agent.Version, response["version"])
	assert.Equal(t, agent.Status, response["status"])
	assert.Equal(t, agent.LastBackupStatus, response["last_backup_status"])
	assert.NotNil(t, response["uptime_seconds"])
	assert.NotNil(t, response["last_seen_at"])

	// Verify timestamps are RFC3339
	lastSeenAt, ok := response["last_seen_at"].(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, lastSeenAt)
	assert.NoError(t, err)

	// Verify disk data is present
	assert.Contains(t, response, "free_disk")
	freeDisk := response["free_disk"].(map[string]interface{})
	disks := freeDisk["disks"].([]interface{})
	assert.Len(t, disks, 2)

	// Verify total_free_bytes calculation
	assert.Contains(t, response, "total_free_bytes")
	totalFree, ok := response["total_free_bytes"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(250000000000), totalFree) // 50GB + 200GB
}

// TestGetAgentByID404 tests 404 for non-existent agent (TDD)
func TestGetAgentByID404(t *testing.T) {
	api, st := setupTestAPI(t)

	// Use an agent ID that doesn't exist but has the correct tenant
	agent := createTestAgent(t, st, "temp-agent")
	tempID := agent.ID
	st.GetDB().Delete(agent)

	req := httptest.NewRequest("GET", fmt.Sprintf("/agents/%s", tempID), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetAgentsTotalFreeBytesCalculation tests the totalFreeBytes aggregation (TDD)
func TestGetAgentsTotalFreeBytesCalculation(t *testing.T) {
	api, st := setupTestAPI(t)

	agent := createTestAgent(t, st, "total-bytes-test")

	// Set up disks with known sizes
	agent.FreeDisk = store.JSONB{
		"disks": []map[string]interface{}{
			{"mountPath": "/", "freeBytes": 10000000000, "totalBytes": 50000000000},      // 10GB free
			{"mountPath": "/data", "freeBytes": 25000000000, "totalBytes": 100000000000}, // 25GB free
			{"mountPath": "/backup", "freeBytes": 5000000000, "totalBytes": 20000000000}, // 5GB free
		},
	}
	st.GetDB().Save(agent)

	req := httptest.NewRequest("GET", fmt.Sprintf("/agents/%s", agent.ID), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	totalFree, ok := response["total_free_bytes"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(40000000000), totalFree) // 10GB + 25GB + 5GB
}

// TestGetAgentsAuthRequired tests authentication requirement (TDD)
func TestGetAgentsAuthRequired(t *testing.T) {
	api, _ := setupTestAPI(t)

	tests := []struct {
		name     string
		endpoint string
	}{
		{"List agents", "/agents"},
		{"Get agent by ID", "/agents/550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			// No auth header
			w := httptest.NewRecorder()

			api.Handler().ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestGetAgentsEmptyList tests empty agent list (TDD)
func TestGetAgentsEmptyList(t *testing.T) {
	api, _ := setupTestAPI(t)

	req := httptest.NewRequest("GET", "/agents", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	agents, ok := response["agents"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, agents)

	total, ok := response["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(0), total)
}
