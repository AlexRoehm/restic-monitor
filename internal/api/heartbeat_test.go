package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeartbeatRouterMapping tests that the router exposes /agents/{id}/heartbeat endpoint (TDD)
func TestHeartbeatRouterMapping(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create a test agent first
	agent := createTestAgent(t, st, "heartbeat-router-test")

	req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", agent.ID), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Route /agents/{id}/heartbeat should exist")
}

// TestHeartbeatValidation tests request validation (TDD)
func TestHeartbeatValidation(t *testing.T) {
	api, st := setupTestAPI(t)

	// Create a test agent
	agent := createTestAgent(t, st, "heartbeat-validation-test")

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Missing version",
			body:           map[string]interface{}{"os": "linux"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "version",
		},
		{
			name:           "Missing os",
			body:           map[string]interface{}{"version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "os",
		},
		{
			name:           "Empty version",
			body:           map[string]interface{}{"version": "", "os": "linux"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "version",
		},
		{
			name:           "Invalid lastBackupStatus",
			body:           map[string]interface{}{"version": "1.0.0", "os": "linux", "lastBackupStatus": "invalid"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "lastBackupStatus",
		},
		{
			name:           "Negative uptimeSeconds",
			body:           map[string]interface{}{"version": "1.0.0", "os": "linux", "uptimeSeconds": -100},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "uptimeSeconds",
		},
		{
			name: "Missing disk mountPath",
			body: map[string]interface{}{
				"version": "1.0.0",
				"os":      "linux",
				"disks": []map[string]interface{}{
					{"freeBytes": 1000, "totalBytes": 2000},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "mountPath",
		},
		{
			name: "Negative disk freeBytes",
			body: map[string]interface{}{
				"version": "1.0.0",
				"os":      "linux",
				"disks": []map[string]interface{}{
					{"mountPath": "/", "freeBytes": -1000, "totalBytes": 2000},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "freeBytes",
		},
		{
			name: "Invalid disk totalBytes",
			body: map[string]interface{}{
				"version": "1.0.0",
				"os":      "linux",
				"disks": []map[string]interface{}{
					{"mountPath": "/", "freeBytes": 1000, "totalBytes": 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "totalBytes",
		},
		{
			name:           "Malformed JSON",
			body:           nil, // Will send invalid JSON
			expectedStatus: http.StatusBadRequest,
			errorContains:  "Invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			} else {
				body = []byte("{invalid json")
			}

			req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", agent.ID), bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			api.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.errorContains != "" {
				errorMsg := ""
				if err, ok := response["error"].(string); ok {
					errorMsg += err
				}
				if details, ok := response["details"].(string); ok {
					errorMsg += details
				}
				assert.Contains(t, errorMsg, tt.errorContains)
			}
		})
	}
}

// TestHeartbeatDBUpdate tests database field updates (TDD)
func TestHeartbeatDBUpdate(t *testing.T) {
	api, st := setupTestAPI(t)

	t.Run("Updates all agent fields", func(t *testing.T) {
		agent := createTestAgent(t, st, "heartbeat-db-update-test")

		// Set initial values that should be updated
		agent.Version = "0.9.0"
		agent.OS = "windows"
		agent.Status = "offline"
		agent.LastBackupStatus = "none"
		st.GetDB().Save(&agent)

		body := map[string]interface{}{
			"version":          "1.2.3",
			"os":               "linux",
			"uptimeSeconds":    864000,
			"lastBackupStatus": "success",
			"disks": []map[string]interface{}{
				{
					"mountPath":  "/",
					"freeBytes":  50000000000,
					"totalBytes": 100000000000,
				},
				{
					"mountPath":  "/data",
					"freeBytes":  200000000000,
					"totalBytes": 500000000000,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", agent.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		// Verify agent was updated in database
		var updatedAgent store.Agent
		err := st.GetDB().Where("id = ?", agent.ID).First(&updatedAgent).Error
		require.NoError(t, err)

		assert.Equal(t, "1.2.3", updatedAgent.Version)
		assert.Equal(t, "linux", updatedAgent.OS)
		assert.Equal(t, "online", updatedAgent.Status)
		assert.Equal(t, "success", updatedAgent.LastBackupStatus)
		assert.NotNil(t, updatedAgent.UptimeSeconds)
		assert.Equal(t, int64(864000), *updatedAgent.UptimeSeconds)
		assert.NotNil(t, updatedAgent.LastSeenAt)
		assert.WithinDuration(t, time.Now(), *updatedAgent.LastSeenAt, 2*time.Second)

		// Verify disk data
		assert.NotNil(t, updatedAgent.FreeDisk)
		disks, ok := updatedAgent.FreeDisk["disks"].([]interface{})
		require.True(t, ok, "FreeDisk should contain 'disks' array")
		assert.Len(t, disks, 2)
	})

	t.Run("Minimal heartbeat updates only required fields", func(t *testing.T) {
		agent := createTestAgent(t, st, "heartbeat-minimal-test")

		body := map[string]interface{}{
			"version": "2.0.0",
			"os":      "darwin",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", agent.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		// Verify agent was updated
		var updatedAgent store.Agent
		err := st.GetDB().Where("id = ?", agent.ID).First(&updatedAgent).Error
		require.NoError(t, err)

		assert.Equal(t, "2.0.0", updatedAgent.Version)
		assert.Equal(t, "darwin", updatedAgent.OS)
		assert.Equal(t, "online", updatedAgent.Status)
		assert.NotNil(t, updatedAgent.LastSeenAt)
	})
}

// TestHeartbeat404InvalidAgent tests 404 for non-existent agents (TDD)
func TestHeartbeat404InvalidAgent(t *testing.T) {
	api, _ := setupTestAPI(t)

	fakeAgentID := uuid.New()
	body := map[string]interface{}{
		"version": "1.0.0",
		"os":      "linux",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", fakeAgentID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	errorMsg := ""
	if err, ok := response["error"].(string); ok {
		errorMsg = err
	}
	assert.Contains(t, errorMsg, "not found")
}

// TestHeartbeatResponseSchema tests response format (TDD)
func TestHeartbeatResponseSchema(t *testing.T) {
	api, st := setupTestAPI(t)

	agent := createTestAgent(t, st, "heartbeat-response-test")

	body := map[string]interface{}{
		"version": "1.0.0",
		"os":      "linux",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", fmt.Sprintf("/agents/%s/heartbeat", agent.ID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify required fields exist
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "nextTaskCheckAfterSeconds")

	// Verify field values
	assert.Equal(t, "ok", response["status"])

	nextCheck, ok := response["nextTaskCheckAfterSeconds"].(float64)
	require.True(t, ok, "nextTaskCheckAfterSeconds should be a number")
	assert.Greater(t, nextCheck, float64(0))
}

// Helper function to create a test agent
func createTestAgent(t *testing.T, st *store.Store, hostname string) *store.Agent {
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: hostname,
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "offline",
	}

	err := st.GetDB().Create(agent).Error
	require.NoError(t, err)

	return agent
}
