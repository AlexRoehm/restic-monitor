package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// TaskTypeCount represents the count of running tasks by type
type TaskTypeCount struct {
	TaskType string `json:"taskType"`
	Count    int    `json:"count"`
}

// TaskTypeCapacity represents available capacity by task type
type TaskTypeCapacity struct {
	TaskType  string `json:"taskType"`
	Available int    `json:"available"`
	Maximum   int    `json:"maximum"`
}

// HeartbeatPayload represents the data sent to the orchestrator in a heartbeat
type HeartbeatPayload struct {
	Version       string     `json:"version"`        // Changed from agentVersion to match API
	OS            string     `json:"os"`             // Changed from platform to match API
	Arch          string     `json:"arch,omitempty"` // Changed from architecture to match API
	UptimeSeconds int64      `json:"uptimeSeconds"`
	DiskUsageMB   int64      `json:"diskUsageMB,omitempty"`
	LastBackupAt  *time.Time `json:"lastBackupAt,omitempty"`
	HeartbeatAt   time.Time  `json:"heartbeatAt"`
	// Load information (EPIC 15 Phase 3)
	CurrentTasksCount    *int               `json:"currentTasksCount,omitempty"`
	RunningTaskTypes     []TaskTypeCount    `json:"runningTaskTypes,omitempty"`
	AvailableSlots       *int               `json:"availableSlots,omitempty"`
	AvailableSlotsByType []TaskTypeCapacity `json:"availableSlotsByType,omitempty"`
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
		Version:       hc.agentVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		UptimeSeconds: int64(time.Since(hc.startTime).Seconds()),
		HeartbeatAt:   time.Now(),
	}

	// Marshal to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	log.Printf("[DEBUG] Heartbeat payload: %s", string(payloadBytes))

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

// BuildHeartbeatPayloadWithLoad creates a heartbeat payload with load information
func BuildHeartbeatPayloadWithLoad(executor *TaskExecutor, version string, uptimeSeconds int64) *HeartbeatPayload {
	payload := &HeartbeatPayload{
		Version:       version,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		UptimeSeconds: uptimeSeconds,
		HeartbeatAt:   time.Now(),
	}

	// Only include load information if executor has concurrency config
	if executor == nil || executor.config == nil {
		return payload
	}

	// Current tasks count
	currentCount := executor.GetRunningTaskCount()
	payload.CurrentTasksCount = &currentCount

	// Available slots
	availableSlots := executor.GetAvailableSlots()
	payload.AvailableSlots = &availableSlots

	// Running task types breakdown
	executor.mu.RLock()
	typeCounts := make(map[string]int)
	for _, taskType := range executor.runningTasks {
		typeCounts[taskType]++
	}
	executor.mu.RUnlock()

	runningTypes := make([]TaskTypeCount, 0, len(typeCounts))
	for taskType, count := range typeCounts {
		runningTypes = append(runningTypes, TaskTypeCount{
			TaskType: taskType,
			Count:    count,
		})
	}
	payload.RunningTaskTypes = runningTypes

	// Available slots by type
	availableByType := []TaskTypeCapacity{
		{
			TaskType:  "backup",
			Available: executor.GetAvailableSlotsByType("backup"),
			Maximum:   executor.config.MaxConcurrentBackups,
		},
		{
			TaskType:  "check",
			Available: executor.GetAvailableSlotsByType("check"),
			Maximum:   executor.config.MaxConcurrentChecks,
		},
		{
			TaskType:  "prune",
			Available: executor.GetAvailableSlotsByType("prune"),
			Maximum:   executor.config.MaxConcurrentPrunes,
		},
	}
	payload.AvailableSlotsByType = availableByType

	return payload
}
