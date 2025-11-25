package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Task represents a backup task assigned to the agent
type Task struct {
	TaskID          string                 `json:"taskId"`
	PolicyID        string                 `json:"policyId"`
	TaskType        string                 `json:"taskType"` // "backup", "check", "prune"
	Repository      string                 `json:"repository"`
	IncludePaths    map[string]interface{} `json:"includePaths,omitempty"`
	ExcludePaths    map[string]interface{} `json:"excludePaths,omitempty"`
	Retention       map[string]interface{} `json:"retention,omitempty"`
	ExecutionParams map[string]interface{} `json:"executionParams,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	ScheduledFor    time.Time              `json:"scheduledFor,omitempty"`
}

// TasksResponse represents the response from GET /agents/{id}/tasks
type TasksResponse struct {
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
}

// TaskClient handles fetching tasks from the orchestrator
type TaskClient struct {
	config     *Config
	state      *State
	httpClient *http.Client
}

// NewTaskClient creates a new task client
func NewTaskClient(cfg *Config, state *State) *TaskClient {
	return &TaskClient{
		config:     cfg,
		state:      state,
		httpClient: &http.Client{Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second},
	}
}

// FetchTasks retrieves pending tasks from the orchestrator
// Returns empty slice if no tasks available
// Returns error only if all retry attempts fail
func (tc *TaskClient) FetchTasks() ([]Task, error) {
	var lastErr error

	for attempt := 0; attempt <= tc.config.RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(tc.config.RetryBackoffSeconds*attempt) * time.Second
			time.Sleep(backoffDuration)
		}

		tasks, err := tc.fetchTasksOnce()
		if err == nil {
			return tasks, nil // Success
		}

		lastErr = err
		// Continue to next retry attempt
	}

	return nil, fmt.Errorf("fetch tasks failed after %d attempts: %w", tc.config.RetryMaxAttempts+1, lastErr)
}

// fetchTasksOnce performs a single task fetch attempt without retry
func (tc *TaskClient) fetchTasksOnce() ([]Task, error) {
	// Validate agent ID
	if _, err := uuid.Parse(tc.state.AgentID); err != nil {
		return nil, fmt.Errorf("invalid agent ID: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/agents/%s/tasks", tc.config.OrchestratorURL, tc.state.AgentID)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create task request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+tc.config.AuthenticationToken)

	// Send request
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusNoContent {
		// No tasks available - this is not an error
		return []Task{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch tasks failed: HTTP %d", resp.StatusCode)
	}

	// Parse response
	var tasksResp TasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tasksResp); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %w", err)
	}

	// Validate tasks
	for i, task := range tasksResp.Tasks {
		if err := validateTask(&task); err != nil {
			return nil, fmt.Errorf("invalid task at index %d: %w", i, err)
		}
	}

	return tasksResp.Tasks, nil
}

// validateTask validates a task structure
func validateTask(task *Task) error {
	// Validate taskId is a valid UUID
	if _, err := uuid.Parse(task.TaskID); err != nil {
		return fmt.Errorf("taskId must be a valid UUID: %w", err)
	}

	// Validate policyId is a valid UUID
	if _, err := uuid.Parse(task.PolicyID); err != nil {
		return fmt.Errorf("policyId must be a valid UUID: %w", err)
	}

	// Validate taskType
	validTypes := []string{"backup", "check", "prune"}
	valid := false
	for _, t := range validTypes {
		if task.TaskType == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("taskType must be one of: backup, check, prune (got: %s)", task.TaskType)
	}

	// Validate repository is non-empty
	if task.Repository == "" {
		return fmt.Errorf("repository cannot be empty")
	}

	// Validate createdAt is set
	if task.CreatedAt.IsZero() {
		return fmt.Errorf("createdAt must be set")
	}

	return nil
}
