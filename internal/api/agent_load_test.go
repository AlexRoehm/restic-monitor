package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatWithLoadInformation(t *testing.T) {
	// Create store with in-memory SQLite
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken: "test-token",
	}
	api := New(cfg, st, nil, "")

	// Create test agent
	agent := store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "offline",
	}
	err = st.GetDB().Create(&agent).Error
	require.NoError(t, err)

	t.Run("Accept heartbeat with load information", func(t *testing.T) {
		currentTasks := 2
		availableSlots := 3

		payload := AgentHeartbeatRequest{
			Version:           "1.0.0",
			OS:                "linux",
			CurrentTasksCount: &currentTasks,
			AvailableSlots:    &availableSlots,
			RunningTaskTypes: []TaskTypeCount{
				{TaskType: "backup", Count: 1},
				{TaskType: "check", Count: 1},
			},
			AvailableSlotsByType: []TaskTypeCapacity{
				{TaskType: "backup", Available: 1, Maximum: 2},
				{TaskType: "check", Available: 0, Maximum: 1},
				{TaskType: "prune", Available: 1, Maximum: 1},
			},
		}

		body, _ := json.Marshal(payload)
		url := "/agents/" + agent.ID.String() + "/heartbeat"
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		api.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify agent was updated
		var updatedAgent store.Agent
		err := st.GetDB().Where("id = ?", agent.ID).First(&updatedAgent).Error
		require.NoError(t, err)
		assert.Equal(t, "online", updatedAgent.Status)
		assert.NotNil(t, updatedAgent.LastSeenAt)
	})

	t.Run("Heartbeat without load information still works", func(t *testing.T) {
		payload := AgentHeartbeatRequest{
			Version: "1.0.0",
			OS:      "linux",
		}

		body, _ := json.Marshal(payload)
		url := "/agents/" + agent.ID.String() + "/heartbeat"
		req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		api.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAgentLoadEndpoint(t *testing.T) {
	// Create store with in-memory SQLite
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken: "test-token",
	}
	api := New(cfg, st, nil, "")

	// Create test agent
	agent := store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	err = st.GetDB().Create(&agent).Error
	require.NoError(t, err)

	t.Run("Get agent load status", func(t *testing.T) {
		url := "/agents/" + agent.ID.String() + "/load"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		api.handleGetAgentLoad(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AgentLoadResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, agent.ID.String(), response.AgentID)
		assert.Equal(t, "test-agent", response.Hostname)
		assert.Equal(t, "online", response.Status)
	})

	t.Run("Agent not found", func(t *testing.T) {
		url := "/agents/" + uuid.New().String() + "/load"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		api.handleGetAgentLoad(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid agent ID", func(t *testing.T) {
		url := "/agents/invalid-uuid/load"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		api.handleGetAgentLoad(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
