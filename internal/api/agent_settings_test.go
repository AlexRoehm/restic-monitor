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

func TestUpdateAgentSettings(t *testing.T) {
	// Create store with in-memory SQLite (migrations run automatically)
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
	assert.NoError(t, err)

	t.Run("Update all settings", func(t *testing.T) {
		maxTasks := 5
		maxBackups := 2
		maxChecks := 2
		maxPrunes := 1
		cpuQuota := 75
		bandwidth := 100

		req := AgentSettingsUpdateRequest{
			MaxConcurrentTasks:   &maxTasks,
			MaxConcurrentBackups: &maxBackups,
			MaxConcurrentChecks:  &maxChecks,
			MaxConcurrentPrunes:  &maxPrunes,
			CPUQuotaPercent:      &cpuQuota,
			BandwidthLimitMbps:   &bandwidth,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AgentSettingsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, 5, response.MaxConcurrentTasks)
		assert.Equal(t, 2, response.MaxConcurrentBackups)
		assert.Equal(t, 2, response.MaxConcurrentChecks)
		assert.Equal(t, 1, response.MaxConcurrentPrunes)
		assert.Equal(t, 75, response.CPUQuotaPercent)
		assert.NotNil(t, response.BandwidthLimitMbps)
		assert.Equal(t, 100, *response.BandwidthLimitMbps)

		// Verify database
		var updatedAgent store.Agent
		err = st.GetDB().Where("id = ?", agent.ID).First(&updatedAgent).Error
		assert.NoError(t, err)
		assert.NotNil(t, updatedAgent.MaxConcurrentTasks)
		assert.Equal(t, 5, *updatedAgent.MaxConcurrentTasks)
	})

	t.Run("Update partial settings", func(t *testing.T) {
		cpuQuota := 60

		req := AgentSettingsUpdateRequest{
			CPUQuotaPercent: &cpuQuota,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		var response AgentSettingsResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, 60, response.CPUQuotaPercent)
	})

	t.Run("Invalid agent ID", func(t *testing.T) {
		maxTasks := 3
		req := AgentSettingsUpdateRequest{
			MaxConcurrentTasks: &maxTasks,
		}

		body, _ := json.Marshal(req)
		url := "/agents/invalid-uuid/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Agent not found", func(t *testing.T) {
		maxTasks := 3
		req := AgentSettingsUpdateRequest{
			MaxConcurrentTasks: &maxTasks,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + uuid.New().String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid max concurrent tasks - negative", func(t *testing.T) {
		maxTasks := -1
		req := AgentSettingsUpdateRequest{
			MaxConcurrentTasks: &maxTasks,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid max concurrent tasks - too high", func(t *testing.T) {
		maxTasks := 101
		req := AgentSettingsUpdateRequest{
			MaxConcurrentTasks: &maxTasks,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid CPU quota - too low", func(t *testing.T) {
		cpuQuota := 0
		req := AgentSettingsUpdateRequest{
			CPUQuotaPercent: &cpuQuota,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid bandwidth limit - negative", func(t *testing.T) {
		bandwidth := -1
		req := AgentSettingsUpdateRequest{
			BandwidthLimitMbps: &bandwidth,
		}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("No updates provided", func(t *testing.T) {
		req := AgentSettingsUpdateRequest{}

		body, _ := json.Marshal(req)
		url := "/agents/" + agent.ID.String() + "/settings"
		r := httptest.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		api.handleUpdateAgentSettings(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
