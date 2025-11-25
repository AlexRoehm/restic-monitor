package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
)

// TestHeartbeatSuccess tests successful heartbeat sending (TDD - Epic 9.2)
func TestHeartbeatSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/agents/550e8400-e29b-41d4-a716-446655440000/heartbeat" {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		// Verify payload
		var payload agent.HeartbeatPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode heartbeat payload: %v", err)
		}

		if payload.AgentVersion != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %s", payload.AgentVersion)
		}
		if payload.Platform == "" {
			t.Error("Expected platform to be set")
		}
		if payload.Architecture == "" {
			t.Error("Expected architecture to be set")
		}
		if payload.UptimeSeconds < 0 {
			t.Error("Expected uptime to be non-negative")
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(agent.HeartbeatResponse{
			Status:  "ok",
			Message: "Heartbeat received",
		})
	}))
	defer server.Close()

	// Create config
	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1,
	}

	// Create state
	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	// Create client and send heartbeat
	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	err := client.SendHeartbeat()
	if err != nil {
		t.Fatalf("Expected heartbeat to succeed, got error: %v", err)
	}

	// Verify state was updated
	if state.LastHeartbeat.Before(time.Now().Add(-5 * time.Second)) {
		t.Error("Expected LastHeartbeat to be updated")
	}
}

// TestHeartbeatInvalidAgentID tests heartbeat with invalid agent ID (TDD - Epic 9.2)
func TestHeartbeatInvalidAgentID(t *testing.T) {
	cfg := &agent.Config{
		OrchestratorURL:     "http://localhost:8080",
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    0,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "invalid-uuid",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	err := client.SendHeartbeat()
	if err == nil {
		t.Fatal("Expected error for invalid agent ID")
	}
}

// TestHeartbeatServerError tests heartbeat with server error (TDD - Epic 9.2)
func TestHeartbeatServerError(t *testing.T) {
	// Create mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(agent.HeartbeatResponse{
			Status:  "error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    0, // No retries for this test
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	err := client.SendHeartbeat()
	if err == nil {
		t.Fatal("Expected error for server error")
	}
	// Error message includes retry attempt count
	if err.Error() != "heartbeat failed after 1 attempts: heartbeat failed: HTTP 500 - Internal server error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestHeartbeatNetworkError tests heartbeat with network error (TDD - Epic 9.2)
func TestHeartbeatNetworkError(t *testing.T) {
	cfg := &agent.Config{
		OrchestratorURL:     "http://localhost:99999", // Invalid port
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  1,
		RetryMaxAttempts:    0, // No retries for this test
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	err := client.SendHeartbeat()
	if err == nil {
		t.Fatal("Expected error for network error")
	}
}

// TestHeartbeatRetrySuccess tests heartbeat retry mechanism (TDD - Epic 9.2)
func TestHeartbeatRetrySuccess(t *testing.T) {
	attemptCount := 0

	// Create mock server that fails twice, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(agent.HeartbeatResponse{Status: "ok"})
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1, // 1 second for testing
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")

	startTime := time.Now()
	err := client.SendHeartbeat()
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Expected heartbeat to succeed after retries, got error: %v", err)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
	// Should have waited: 0s (attempt 1) + 1s (backoff) + 2s (backoff) = ~3s
	if duration < 2*time.Second {
		t.Errorf("Expected backoff delay, but took only %v", duration)
	}
}

// TestHeartbeatMaxRetriesExceeded tests heartbeat when max retries exceeded (TDD - Epic 9.2)
func TestHeartbeatMaxRetriesExceeded(t *testing.T) {
	attemptCount := 0

	// Create mock server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    2,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	err := client.SendHeartbeat()

	if err == nil {
		t.Fatal("Expected error when max retries exceeded")
	}
	// Should attempt: initial + 2 retries = 3 total attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attemptCount)
	}
}

// TestHeartbeatPayloadStructure tests the heartbeat payload structure (TDD - Epic 9.2)
func TestHeartbeatPayloadStructure(t *testing.T) {
	var capturedPayload agent.HeartbeatPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    0,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "2.5.1")

	// Wait a bit to get non-zero uptime
	time.Sleep(100 * time.Millisecond)

	client.SendHeartbeat()

	// Verify payload fields
	if capturedPayload.AgentVersion != "2.5.1" {
		t.Errorf("Expected version 2.5.1, got %s", capturedPayload.AgentVersion)
	}
	if capturedPayload.Platform == "" {
		t.Error("Expected platform to be set")
	}
	if capturedPayload.Architecture == "" {
		t.Error("Expected architecture to be set")
	}
	if capturedPayload.UptimeSeconds < 0 {
		t.Error("Expected uptime to be non-negative")
	}
	if capturedPayload.HeartbeatAt.IsZero() {
		t.Error("Expected heartbeatAt to be set")
	}
}

// TestHeartbeatAuthorizationHeader tests that authorization header is sent (TDD - Epic 9.2)
func TestHeartbeatAuthorizationHeader(t *testing.T) {
	authHeaderReceived := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderReceived = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "my-secret-token-123",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    0,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewHeartbeatClient(cfg, state, "1.0.0")
	client.SendHeartbeat()

	expectedHeader := "Bearer my-secret-token-123"
	if authHeaderReceived != expectedHeader {
		t.Errorf("Expected Authorization header %q, got %q", expectedHeader, authHeaderReceived)
	}
}
