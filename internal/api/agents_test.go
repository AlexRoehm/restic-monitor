package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentRegistrationRouterMapping tests that the router exposes /agents/register endpoint (TDD)
func TestAgentRegistrationRouterMapping(t *testing.T) {
	api, _ := setupTestAPI(t)

	req := httptest.NewRequest("POST", "/agents/register", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, w.Code, "Route /agents/register should exist")
}

// TestAgentRegistrationValidation tests request validation (TDD)
func TestAgentRegistrationValidation(t *testing.T) {
	api, _ := setupTestAPI(t)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Missing hostname",
			body:           map[string]interface{}{"os": "linux", "arch": "amd64", "version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "hostname",
		},
		{
			name:           "Empty hostname",
			body:           map[string]interface{}{"hostname": "", "os": "linux", "arch": "amd64", "version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "hostname",
		},
		{
			name:           "Missing os",
			body:           map[string]interface{}{"hostname": "test-host", "arch": "amd64", "version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "os",
		},
		{
			name:           "Missing arch",
			body:           map[string]interface{}{"hostname": "test-host", "os": "linux", "version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "arch",
		},
		{
			name:           "Missing version",
			body:           map[string]interface{}{"hostname": "test-host", "os": "linux", "arch": "amd64"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "version",
		},
		{
			name:           "Hostname too long",
			body:           map[string]interface{}{"hostname": string(make([]byte, 300)), "os": "linux", "arch": "amd64", "version": "1.0.0"},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "hostname",
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

			req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(body))
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

// TestAgentRegistrationDBWrite tests database persistence (TDD)
func TestAgentRegistrationDBWrite(t *testing.T) {
	api, st := setupTestAPI(t)

	t.Run("Creates new agent", func(t *testing.T) {
		body := map[string]interface{}{
			"hostname": "web-server-01.example.com",
			"os":       "linux",
			"arch":     "amd64",
			"version":  "1.0.0",
			"ip":       "192.168.1.100",
			"metadata": map[string]interface{}{
				"restic_version": "0.16.2",
				"cpu_cores":      float64(8),
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		// Verify agent was created in database
		var agent store.Agent
		err := st.GetDB().Where("hostname = ?", "web-server-01.example.com").First(&agent).Error
		require.NoError(t, err)
		assert.Equal(t, "linux", agent.OS)
		assert.Equal(t, "amd64", agent.Arch)
		assert.Equal(t, "1.0.0", agent.Version)
		assert.Equal(t, "online", agent.Status)
		assert.NotNil(t, agent.Metadata)
		assert.Equal(t, "0.16.2", agent.Metadata["restic_version"])
	})

	t.Run("Updates existing agent on duplicate hostname", func(t *testing.T) {
		// First registration
		body1 := map[string]interface{}{
			"hostname": "update-test.example.com",
			"os":       "linux",
			"arch":     "amd64",
			"version":  "1.0.0",
		}
		bodyBytes1, _ := json.Marshal(body1)

		req1 := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(bodyBytes1))
		req1.Header.Set("Authorization", "Bearer test-token")
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		api.Handler().ServeHTTP(w1, req1)
		require.Equal(t, http.StatusCreated, w1.Code)

		var response1 map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &response1)
		firstAgentID := response1["agentId"].(string)

		// Second registration with same hostname
		body2 := map[string]interface{}{
			"hostname": "update-test.example.com",
			"os":       "linux",
			"arch":     "amd64",
			"version":  "2.0.0", // Updated version
		}
		bodyBytes2, _ := json.Marshal(body2)

		req2 := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(bodyBytes2))
		req2.Header.Set("Authorization", "Bearer test-token")
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		api.Handler().ServeHTTP(w2, req2)

		require.Equal(t, http.StatusOK, w2.Code) // 200 for update

		var response2 map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &response2)
		secondAgentID := response2["agentId"].(string)

		// Should return same agent ID
		assert.Equal(t, firstAgentID, secondAgentID)

		// Verify only one agent exists
		var count int64
		st.GetDB().Model(&store.Agent{}).Where("hostname = ?", "update-test.example.com").Count(&count)
		assert.Equal(t, int64(1), count)

		// Verify version was updated
		var agent store.Agent
		st.GetDB().Where("hostname = ?", "update-test.example.com").First(&agent)
		assert.Equal(t, "2.0.0", agent.Version)
	})
}

// TestAgentRegistrationResponseSchema tests response format (TDD)
func TestAgentRegistrationResponseSchema(t *testing.T) {
	api, _ := setupTestAPI(t)

	body := map[string]interface{}{
		"hostname": "schema-test.example.com",
		"os":       "linux",
		"arch":     "amd64",
		"version":  "1.0.0",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all required fields exist
	assert.Contains(t, response, "agentId")
	assert.Contains(t, response, "hostname")
	assert.Contains(t, response, "registeredAt")
	assert.Contains(t, response, "updatedAt")
	assert.Contains(t, response, "message")

	// Verify field types
	agentID, ok := response["agentId"].(string)
	require.True(t, ok, "agentId should be string")
	_, err = uuid.Parse(agentID)
	assert.NoError(t, err, "agentId should be valid UUID")

	assert.Equal(t, "schema-test.example.com", response["hostname"])

	// Verify timestamps are RFC3339
	registeredAt, ok := response["registeredAt"].(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, registeredAt)
	assert.NoError(t, err, "registeredAt should be RFC3339 format")

	updatedAt, ok := response["updatedAt"].(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, updatedAt)
	assert.NoError(t, err, "updatedAt should be RFC3339 format")

	message, ok := response["message"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, message)
}

// TestAgentRegistrationAuth tests authentication requirement (TDD)
func TestAgentRegistrationAuth(t *testing.T) {
	api, _ := setupTestAPI(t)

	body := map[string]interface{}{
		"hostname": "auth-test.example.com",
		"os":       "linux",
		"arch":     "amd64",
		"version":  "1.0.0",
	}
	bodyBytes, _ := json.Marshal(body)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid token",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/agents/register", bytes.NewReader(bodyBytes))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			api.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// setupTestAPI creates a test API instance with in-memory database
func setupTestAPI(t *testing.T) (*API, *store.Store) {
	// Create store with in-memory SQLite (migrations run automatically)
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken: "test-token",
	}

	api := New(cfg, st, nil, "")

	return api, st
}
