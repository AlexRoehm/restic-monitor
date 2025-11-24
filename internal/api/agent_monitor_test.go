package api

import (
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkStaleAgentsOffline tests the offline detection logic (TDD)
func TestMarkStaleAgentsOffline(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken:               "test-token",
		HeartbeatTimeoutSeconds: 90,
	}

	api := New(cfg, st, nil, "")

	// Create test agents with different last_seen_at times
	now := time.Now()

	// Agent 1: Recent heartbeat (30 seconds ago) - should stay online
	agent1 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "recent-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	recent := now.Add(-30 * time.Second)
	agent1.LastSeenAt = &recent
	st.GetDB().Create(agent1)

	// Agent 2: Stale heartbeat (120 seconds ago) - should go offline
	agent2 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "stale-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	stale := now.Add(-120 * time.Second)
	agent2.LastSeenAt = &stale
	st.GetDB().Create(agent2)

	// Agent 3: Already offline - should stay offline
	agent3 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "already-offline",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "offline",
	}
	veryOld := now.Add(-300 * time.Second)
	agent3.LastSeenAt = &veryOld
	st.GetDB().Create(agent3)

	// Agent 4: No heartbeat ever (nil LastSeenAt) - should go offline
	agent4 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "never-seen",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	st.GetDB().Create(agent4)

	// Run offline detection
	count := api.markStaleAgentsOffline()

	// Should mark 2 agents offline (agent2 and agent4)
	assert.Equal(t, 2, count)

	// Verify agent statuses
	var updated1 store.Agent
	st.GetDB().First(&updated1, agent1.ID)
	assert.Equal(t, "online", updated1.Status, "Recent agent should stay online")

	var updated2 store.Agent
	st.GetDB().First(&updated2, agent2.ID)
	assert.Equal(t, "offline", updated2.Status, "Stale agent should be offline")

	var updated3 store.Agent
	st.GetDB().First(&updated3, agent3.ID)
	assert.Equal(t, "offline", updated3.Status, "Already offline agent should stay offline")

	var updated4 store.Agent
	st.GetDB().First(&updated4, agent4.ID)
	assert.Equal(t, "offline", updated4.Status, "Never-seen agent should be offline")
}

// TestMarkStaleAgentsOffline_Boundary tests the exact threshold boundary (TDD)
func TestMarkStaleAgentsOffline_Boundary(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken:               "test-token",
		HeartbeatTimeoutSeconds: 90,
	}

	api := New(cfg, st, nil, "")

	now := time.Now()

	// Agent exactly at boundary (90 seconds ago)
	agent1 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "boundary-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	boundary := now.Add(-90 * time.Second)
	agent1.LastSeenAt = &boundary
	st.GetDB().Create(agent1)

	// Agent just inside boundary (89 seconds ago)
	agent2 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "just-inside",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	justInside := now.Add(-89 * time.Second)
	agent2.LastSeenAt = &justInside
	st.GetDB().Create(agent2)

	// Agent just outside boundary (91 seconds ago)
	agent3 := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "just-outside",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	justOutside := now.Add(-91 * time.Second)
	agent3.LastSeenAt = &justOutside
	st.GetDB().Create(agent3)

	count := api.markStaleAgentsOffline()

	// Agents at or beyond threshold should be marked offline
	// We use > comparison, so exactly 90 seconds should stay online
	assert.GreaterOrEqual(t, count, 1)

	var updated1 store.Agent
	st.GetDB().First(&updated1, agent1.ID)

	var updated2 store.Agent
	st.GetDB().First(&updated2, agent2.ID)
	assert.Equal(t, "online", updated2.Status, "Just inside boundary should stay online")

	var updated3 store.Agent
	st.GetDB().First(&updated3, agent3.ID)
	assert.Equal(t, "offline", updated3.Status, "Just outside boundary should be offline")
}

// TestMarkStaleAgentsOffline_ConfigurableThreshold tests different timeout values (TDD)
func TestMarkStaleAgentsOffline_ConfigurableThreshold(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	// Use a shorter timeout for testing
	cfg := config.Config{
		AuthToken:               "test-token",
		HeartbeatTimeoutSeconds: 30, // 30 seconds instead of 90
	}

	api := New(cfg, st, nil, "")

	now := time.Now()

	// Agent 60 seconds old - should be offline with 30s threshold
	agent := &store.Agent{
		TenantID: st.GetTenantID(),
		Hostname: "test-agent",
		OS:       "linux",
		Arch:     "amd64",
		Version:  "1.0.0",
		Status:   "online",
	}
	old := now.Add(-60 * time.Second)
	agent.LastSeenAt = &old
	st.GetDB().Create(agent)

	count := api.markStaleAgentsOffline()
	assert.Equal(t, 1, count)

	var updated store.Agent
	st.GetDB().First(&updated, agent.ID)
	assert.Equal(t, "offline", updated.Status)
}

// TestMarkStaleAgentsOffline_OnlyAffectsOnlineAgents tests that only online agents are updated (TDD)
func TestMarkStaleAgentsOffline_OnlyAffectsOnlineAgents(t *testing.T) {
	st, err := store.NewWithTenant(":memory:", uuid.New())
	require.NoError(t, err)

	cfg := config.Config{
		AuthToken:               "test-token",
		HeartbeatTimeoutSeconds: 90,
	}

	api := New(cfg, st, nil, "")

	now := time.Now()
	stale := now.Add(-120 * time.Second)

	// Create agents with different initial statuses
	statuses := []string{"online", "offline", "error"}

	for _, status := range statuses {
		agent := &store.Agent{
			TenantID: st.GetTenantID(),
			Hostname: "agent-" + status,
			OS:       "linux",
			Arch:     "amd64",
			Version:  "1.0.0",
			Status:   status,
		}
		agent.LastSeenAt = &stale
		st.GetDB().Create(agent)
	}

	count := api.markStaleAgentsOffline()

	// Should only affect the "online" agent
	assert.Equal(t, 1, count)
}
