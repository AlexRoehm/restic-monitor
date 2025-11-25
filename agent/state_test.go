package agent_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadStateFileNotFound tests that missing state file returns nil state (TDD - Epic 8.2)
func TestLoadStateFileNotFound(t *testing.T) {
	state, err := agent.LoadState("/nonexistent/state.json")
	require.NoError(t, err, "missing state file should not be an error")
	assert.Nil(t, state, "should return nil for non-existent state file")
}

// TestLoadStateEmptyFile tests handling of empty state file (TDD - Epic 8.2)
func TestLoadStateEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create empty file
	err := os.WriteFile(stateFile, []byte(""), 0600)
	require.NoError(t, err)

	state, err := agent.LoadState(stateFile)
	assert.Error(t, err, "empty file should return error")
	assert.Nil(t, state)
}

// TestLoadStateInvalidJSON tests handling of malformed JSON (TDD - Epic 8.2)
func TestLoadStateInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create invalid JSON
	err := os.WriteFile(stateFile, []byte("{invalid json}"), 0600)
	require.NoError(t, err)

	state, err := agent.LoadState(stateFile)
	assert.Error(t, err, "invalid JSON should return error")
	assert.Nil(t, state)
}

// TestLoadStateValid tests loading valid state file (TDD - Epic 8.2)
func TestLoadStateValid(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	stateJSON := `{
		"agentId": "550e8400-e29b-41d4-a716-446655440000",
		"registeredAt": "2025-01-15T10:30:00Z",
		"lastHeartbeat": "2025-01-15T12:45:30Z",
		"hostname": "test-server"
	}`

	err := os.WriteFile(stateFile, []byte(stateJSON), 0600)
	require.NoError(t, err)

	state, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", state.AgentID)
	assert.Equal(t, "test-server", state.Hostname)
	assert.False(t, state.RegisteredAt.IsZero())
	assert.False(t, state.LastHeartbeat.IsZero())
}

// TestSaveStateNew tests creating new state file (TDD - Epic 8.2)
func TestSaveStateNew(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	now := time.Now().UTC()
	state := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  now,
		LastHeartbeat: now,
		Hostname:      "backup-server-01",
	}

	err := agent.SaveState(stateFile, state)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should exist")

	// Verify file permissions are restrictive
	info, err := os.Stat(stateFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "state file should have 0600 permissions")

	// Verify we can load it back
	loaded, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, state.AgentID, loaded.AgentID)
	assert.Equal(t, state.Hostname, loaded.Hostname)
}

// TestSaveStateUpdate tests updating existing state file (TDD - Epic 8.2)
func TestSaveStateUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create initial state
	initialTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	state := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  initialTime,
		LastHeartbeat: initialTime,
		Hostname:      "server-01",
	}

	err := agent.SaveState(stateFile, state)
	require.NoError(t, err)

	// Update state
	newTime := time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)
	state.LastHeartbeat = newTime

	err = agent.SaveState(stateFile, state)
	require.NoError(t, err)

	// Verify update
	loaded, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, initialTime.Unix(), loaded.RegisteredAt.Unix())
	assert.Equal(t, newTime.Unix(), loaded.LastHeartbeat.Unix())
}

// TestSaveStateAtomicWrite tests atomic write operation (TDD - Epic 8.2)
func TestSaveStateAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create initial state
	state1 := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  time.Now().UTC(),
		LastHeartbeat: time.Now().UTC(),
		Hostname:      "server-01",
	}

	err := agent.SaveState(stateFile, state1)
	require.NoError(t, err)

	// Read initial content
	initialContent, err := os.ReadFile(stateFile)
	require.NoError(t, err)

	// Update with new state
	state2 := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  state1.RegisteredAt,
		LastHeartbeat: time.Now().UTC().Add(time.Hour),
		Hostname:      "server-01",
	}

	err = agent.SaveState(stateFile, state2)
	require.NoError(t, err)

	// Verify new content is different
	newContent, err := os.ReadFile(stateFile)
	require.NoError(t, err)
	assert.NotEqual(t, string(initialContent), string(newContent), "file should be updated")

	// Verify we can still load valid state
	loaded, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, state2.AgentID, loaded.AgentID)
}

// TestSaveStateDirectoryCreation tests automatic directory creation (TDD - Epic 8.2)
func TestSaveStateDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "subdir", "nested", "state.json")

	state := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  time.Now().UTC(),
		LastHeartbeat: time.Now().UTC(),
		Hostname:      "server-01",
	}

	err := agent.SaveState(stateFile, state)
	require.NoError(t, err)

	// Verify directories were created
	_, err = os.Stat(filepath.Dir(stateFile))
	assert.NoError(t, err, "parent directories should be created")

	// Verify file exists
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "state file should exist")
}

// TestStateValidation tests state field validation (TDD - Epic 8.2)
func TestStateValidation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	tests := []struct {
		name        string
		state       *agent.State
		shouldError bool
	}{
		{
			name: "valid state",
			state: &agent.State{
				AgentID:       "550e8400-e29b-41d4-a716-446655440000",
				RegisteredAt:  time.Now().UTC(),
				LastHeartbeat: time.Now().UTC(),
				Hostname:      "server-01",
			},
			shouldError: false,
		},
		{
			name: "missing agent ID",
			state: &agent.State{
				RegisteredAt:  time.Now().UTC(),
				LastHeartbeat: time.Now().UTC(),
				Hostname:      "server-01",
			},
			shouldError: true,
		},
		{
			name: "invalid UUID format",
			state: &agent.State{
				AgentID:       "not-a-uuid",
				RegisteredAt:  time.Now().UTC(),
				LastHeartbeat: time.Now().UTC(),
				Hostname:      "server-01",
			},
			shouldError: true,
		},
		{
			name: "missing hostname",
			state: &agent.State{
				AgentID:       "550e8400-e29b-41d4-a716-446655440000",
				RegisteredAt:  time.Now().UTC(),
				LastHeartbeat: time.Now().UTC(),
			},
			shouldError: true,
		},
		{
			name: "zero registered time",
			state: &agent.State{
				AgentID:       "550e8400-e29b-41d4-a716-446655440000",
				RegisteredAt:  time.Time{},
				LastHeartbeat: time.Now().UTC(),
				Hostname:      "server-01",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.SaveState(stateFile, tt.state)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpdateLastHeartbeat tests updating just the heartbeat timestamp (TDD - Epic 8.2)
func TestUpdateLastHeartbeat(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create initial state
	initialTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	state := &agent.State{
		AgentID:       "123e4567-e89b-12d3-a456-426614174000",
		RegisteredAt:  initialTime,
		LastHeartbeat: initialTime,
		Hostname:      "server-01",
	}

	err := agent.SaveState(stateFile, state)
	require.NoError(t, err)

	// Load and update heartbeat
	loaded, err := agent.LoadState(stateFile)
	require.NoError(t, err)

	newHeartbeat := time.Now().UTC()
	loaded.LastHeartbeat = newHeartbeat

	err = agent.SaveState(stateFile, loaded)
	require.NoError(t, err)

	// Verify only heartbeat changed
	final, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, state.AgentID, final.AgentID)
	assert.Equal(t, state.Hostname, final.Hostname)
	assert.Equal(t, initialTime.Unix(), final.RegisteredAt.Unix())
	assert.Equal(t, newHeartbeat.Unix(), final.LastHeartbeat.Unix())
}
