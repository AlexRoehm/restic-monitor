package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// RegistrationRequest represents the agent registration request payload
type RegistrationRequest struct {
	Hostname  string                 `json:"hostname"`
	OS        string                 `json:"os"`
	Arch      string                 `json:"arch"`
	Version   string                 `json:"version"`
	AuthToken string                 `json:"authToken,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RegistrationResponse represents the orchestrator's registration response
type RegistrationResponse struct {
	AgentID   string    `json:"agentId"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Register registers the agent with the orchestrator on first run.
// If the agent is already registered (state file exists with valid agentId),
// this function returns immediately without making an API call.
// On successful registration, the agent state is persisted to the state file.
func Register(cfg *Config) error {
	// Check if already registered
	existingState, err := LoadState(cfg.StateFile)
	if err != nil {
		return fmt.Errorf("failed to load existing state: %w", err)
	}

	if existingState != nil && existingState.AgentID != "" {
		// Already registered, no need to register again
		return nil
	}

	// Determine hostname
	hostname := cfg.HostnameOverride
	if hostname == "" {
		detectedHostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to detect hostname: %w", err)
		}
		hostname = detectedHostname
	}

	// Prepare registration request
	reqBody := RegistrationRequest{
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Version:   "1.0.0", // TODO: Pass this as parameter
		AuthToken: cfg.AuthenticationToken,
	}

	// Register with retry logic
	var resp *RegistrationResponse
	var lastErr error

	maxAttempts := cfg.RetryMaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3 // Default
	}

	backoffSeconds := cfg.RetryBackoffSeconds
	if backoffSeconds == 0 {
		backoffSeconds = 5 // Default
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, lastErr = performRegistration(cfg.OrchestratorURL, reqBody, cfg.HTTPTimeoutSeconds)
		if lastErr == nil {
			// Success!
			break
		}

		// If this was the last attempt, don't sleep
		if attempt < maxAttempts {
			// Exponential backoff
			sleepDuration := time.Duration(backoffSeconds*attempt) * time.Second
			time.Sleep(sleepDuration)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("registration failed after %d attempts (max retries exceeded): %w", maxAttempts, lastErr)
	}

	// Validate response
	if err := validateRegistrationResponse(resp); err != nil {
		return fmt.Errorf("invalid registration response: %w", err)
	}

	// Create and save state
	state := &State{
		AgentID:       resp.AgentID,
		RegisteredAt:  time.Now().UTC(),
		LastHeartbeat: time.Now().UTC(),
		Hostname:      hostname,
	}

	if err := SaveState(cfg.StateFile, state); err != nil {
		return fmt.Errorf("failed to save state after registration: %w", err)
	}

	return nil
}

// performRegistration makes a single registration attempt to the orchestrator
func performRegistration(orchestratorURL string, req RegistrationRequest, timeoutSeconds int) (*RegistrationResponse, error) {
	// Marshal request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := orchestratorURL + "/agents/register"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Create HTTP client with timeout
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds == 0 {
		timeout = 30 * time.Second // Default
	}
	client := &http.Client{
		Timeout: timeout,
	}

	// Make request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if httpResp.StatusCode != http.StatusCreated && httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registration failed with status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var resp RegistrationResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// validateRegistrationResponse validates the registration response
func validateRegistrationResponse(resp *RegistrationResponse) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}

	if resp.AgentID == "" {
		return fmt.Errorf("agentId is empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(resp.AgentID); err != nil {
		return fmt.Errorf("agentId is not a valid UUID: %w", err)
	}

	return nil
}
