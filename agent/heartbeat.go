package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// HeartbeatPayload represents the data sent to the orchestrator in a heartbeat
type HeartbeatPayload struct {
	AgentVersion  string     `json:"agentVersion"`
	Platform      string     `json:"platform"`     // e.g., "linux", "darwin", "windows"
	Architecture  string     `json:"architecture"` // e.g., "amd64", "arm64"
	UptimeSeconds int64      `json:"uptimeSeconds"`
	DiskUsageMB   int64      `json:"diskUsageMB,omitempty"`
	LastBackupAt  *time.Time `json:"lastBackupAt,omitempty"`
	HeartbeatAt   time.Time  `json:"heartbeatAt"`
}

// HeartbeatResponse represents the response from the orchestrator
type HeartbeatResponse struct {
	Status  string `json:"status"` // "ok", "error"
	Message string `json:"message,omitempty"`
}

// HeartbeatClient handles sending heartbeats to the orchestrator
type HeartbeatClient struct {
	config       *Config
	state        *State
	httpClient   *http.Client
	agentVersion string
	startTime    time.Time
}

// NewHeartbeatClient creates a new heartbeat client
func NewHeartbeatClient(cfg *Config, state *State, agentVersion string) *HeartbeatClient {
	return &HeartbeatClient{
		config:       cfg,
		state:        state,
		httpClient:   &http.Client{Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second},
		agentVersion: agentVersion,
		startTime:    time.Now(),
	}
}

// SendHeartbeat sends a heartbeat to the orchestrator
// Returns error only if all retry attempts fail
func (hc *HeartbeatClient) SendHeartbeat() error {
	var lastErr error

	for attempt := 0; attempt <= hc.config.RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(hc.config.RetryBackoffSeconds*attempt) * time.Second
			time.Sleep(backoffDuration)
		}

		err := hc.sendHeartbeatOnce()
		if err == nil {
			return nil // Success
		}

		lastErr = err
		// Continue to next retry attempt
	}

	return fmt.Errorf("heartbeat failed after %d attempts: %w", hc.config.RetryMaxAttempts+1, lastErr)
}

// sendHeartbeatOnce performs a single heartbeat attempt without retry
func (hc *HeartbeatClient) sendHeartbeatOnce() error {
	// Validate agent ID
	if _, err := uuid.Parse(hc.state.AgentID); err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}

	// Build payload
	payload := HeartbeatPayload{
		AgentVersion:  hc.agentVersion,
		Platform:      runtime.GOOS,
		Architecture:  runtime.GOARCH,
		UptimeSeconds: int64(time.Since(hc.startTime).Seconds()),
		HeartbeatAt:   time.Now(),
	}

	// Marshal to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/agents/%s/heartbeat", hc.config.OrchestratorURL, hc.state.AgentID)

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+hc.config.AuthenticationToken)

	// Send request
	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		// Try to parse error response
		var heartbeatResp HeartbeatResponse
		if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err == nil && heartbeatResp.Message != "" {
			return fmt.Errorf("heartbeat failed: HTTP %d - %s", resp.StatusCode, heartbeatResp.Message)
		}
		return fmt.Errorf("heartbeat failed: HTTP %d", resp.StatusCode)
	}

	// Update last heartbeat in state
	hc.state.LastHeartbeat = time.Now()

	return nil
}
