package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterFirstRun tests successful first-time registration (TDD - Epic 8.3)
func TestRegisterFirstRun(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Mock orchestrator server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/agents/register", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var req agent.RegistrationRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Validate request fields
		assert.NotEmpty(t, req.Hostname)
		assert.Equal(t, "test-token", req.AuthToken)

		// Return successful response
		resp := agent.RegistrationResponse{
			AgentID:   "550e8400-e29b-41d4-a716-446655440000",
			Status:    "registered",
			Message:   "Agent registered successfully",
			ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create config
	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
	}

	// Register agent
	err := agent.Register(cfg)
	require.NoError(t, err)

	// Verify state was saved
	state, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", state.AgentID)
	assert.NotEmpty(t, state.Hostname)
	assert.False(t, state.RegisteredAt.IsZero())
}

// TestRegisterAlreadyRegistered tests that already-registered agents skip registration (TDD - Epic 8.3)
func TestRegisterAlreadyRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create existing state
	existingState := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now().UTC().Add(-24 * time.Hour),
		LastHeartbeat: time.Now().UTC(),
		Hostname:      "existing-host",
	}
	err := agent.SaveState(stateFile, existingState)
	require.NoError(t, err)

	// Mock server that should NOT be called
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Create config
	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
	}

	// Register should be a no-op
	err = agent.Register(cfg)
	require.NoError(t, err)

	// Verify server was NOT called
	assert.False(t, serverCalled, "server should not be called for already-registered agent")

	// Verify state unchanged
	state, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", state.AgentID)
}

// TestRegisterInvalidToken tests registration with invalid authentication token (TDD - Epic 8.3)
func TestRegisterInvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Mock server returning 401 Unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid authentication token",
		})
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "invalid-token",
		StateFile:           stateFile,
	}

	err := agent.Register(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")

	// Verify no state file was created
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "state file should not exist after failed registration")
}

// TestRegisterServerError tests registration with server error (TDD - Epic 8.3)
func TestRegisterServerError(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Mock server returning 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "database connection failed",
		})
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
	}

	err := agent.Register(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestRegisterNetworkError tests registration with network failure (TDD - Epic 8.3)
func TestRegisterNetworkError(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	cfg := &agent.Config{
		OrchestratorURL:     "http://localhost:1", // Invalid port
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
		HTTPTimeoutSeconds:  1,
	}

	err := agent.Register(cfg)
	assert.Error(t, err)
}

// TestRegisterWithRetry tests registration retry logic (TDD - Epic 8.3)
func TestRegisterWithRetry(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Track number of attempts
	attemptCount := 0

	// Mock server that fails twice, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount < 3 {
			// First two attempts fail
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Third attempt succeeds
		resp := agent.RegistrationResponse{
			AgentID:   "550e8400-e29b-41d4-a716-446655440000",
			Status:    "registered",
			Message:   "Agent registered successfully",
			ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1,
	}

	err := agent.Register(cfg)
	require.NoError(t, err)

	// Verify it retried the correct number of times
	assert.Equal(t, 3, attemptCount, "should have retried until success")

	// Verify registration succeeded
	state, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", state.AgentID)
}

// TestRegisterMaxRetriesExceeded tests registration failure after max retries (TDD - Epic 8.3)
func TestRegisterMaxRetriesExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Track number of attempts
	attemptCount := 0

	// Mock server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1,
	}

	err := agent.Register(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries")

	// Verify it retried the correct number of times
	assert.Equal(t, 3, attemptCount, "should have retried max times")
}

// TestRegisterRequestStructure tests registration request format (TDD - Epic 8.3)
func TestRegisterRequestStructure(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Capture request
	var capturedRequest agent.RegistrationRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture and validate request
		err := json.NewDecoder(r.Body).Decode(&capturedRequest)
		require.NoError(t, err)

		// Return success
		resp := agent.RegistrationResponse{
			AgentID:   "550e8400-e29b-41d4-a716-446655440000",
			Status:    "registered",
			Message:   "OK",
			ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token-123",
		StateFile:           stateFile,
		HostnameOverride:    "custom-hostname",
	}

	err := agent.Register(cfg)
	require.NoError(t, err)

	// Verify request structure
	assert.Equal(t, "custom-hostname", capturedRequest.Hostname)
	assert.Equal(t, "test-token-123", capturedRequest.AuthToken)
}

// TestRegisterResponseValidation tests response validation (TDD - Epic 8.3)
func TestRegisterResponseValidation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	tests := []struct {
		name        string
		response    agent.RegistrationResponse
		shouldError bool
	}{
		{
			name: "valid response",
			response: agent.RegistrationResponse{
				AgentID:   "550e8400-e29b-41d4-a716-446655440000",
				Status:    "registered",
				Message:   "OK",
				ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
			},
			shouldError: false,
		},
		{
			name: "empty agent ID",
			response: agent.RegistrationResponse{
				AgentID:   "",
				Status:    "registered",
				Message:   "OK",
				ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
			},
			shouldError: true,
		},
		{
			name: "invalid UUID",
			response: agent.RegistrationResponse{
				AgentID:   "not-a-uuid",
				Status:    "registered",
				Message:   "OK",
				ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			cfg := &agent.Config{
				OrchestratorURL:     server.URL,
				AuthenticationToken: "test-token",
				StateFile:           stateFile,
			}

			err := agent.Register(cfg)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Clean up state file between tests
			os.Remove(stateFile)
		})
	}
}

// TestRegisterHostnameDetection tests automatic hostname detection (TDD - Epic 8.3)
func TestRegisterHostnameDetection(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	var capturedHostname string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req agent.RegistrationRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedHostname = req.Hostname

		resp := agent.RegistrationResponse{
			AgentID:   "550e8400-e29b-41d4-a716-446655440000",
			Status:    "registered",
			Message:   "OK",
			ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		StateFile:           stateFile,
		// No HostnameOverride - should detect automatically
	}

	err := agent.Register(cfg)
	require.NoError(t, err)

	// Verify a hostname was detected
	assert.NotEmpty(t, capturedHostname, "hostname should be auto-detected")

	// Verify state contains the same hostname
	state, err := agent.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, capturedHostname, state.Hostname)
}
