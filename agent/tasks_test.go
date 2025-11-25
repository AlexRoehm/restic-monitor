package agent_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
)

// TestFetchTasksSuccess tests successful task fetching (TDD - Epic 9.3)
func TestFetchTasksSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/agents/550e8400-e29b-41d4-a716-446655440000/tasks" {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		// Send success response with tasks
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(agent.TasksResponse{
			Tasks: []agent.Task{
				{
					TaskID:       "123e4567-e89b-12d3-a456-426614174000",
					PolicyID:     "234e5678-e89b-12d3-a456-426614174001",
					TaskType:     "backup",
					Repository:   "s3:bucket/repo",
					CreatedAt:    time.Now(),
					ScheduledFor: time.Now().Add(1 * time.Hour),
				},
			},
			Count: 1,
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

	// Create client and fetch tasks
	client := agent.NewTaskClient(cfg, state)
	tasks, err := client.FetchTasks()
	if err != nil {
		t.Fatalf("Expected task fetch to succeed, got error: %v", err)
	}

	// Verify tasks
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].TaskType != "backup" {
		t.Errorf("Expected taskType 'backup', got %s", tasks[0].TaskType)
	}
	if tasks[0].Repository != "s3:bucket/repo" {
		t.Errorf("Expected repository 's3:bucket/repo', got %s", tasks[0].Repository)
	}
}

// TestFetchTasksEmpty tests fetching when no tasks available (TDD - Epic 9.3)
func TestFetchTasksEmpty(t *testing.T) {
	// Create mock server that returns 204 No Content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
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

	client := agent.NewTaskClient(cfg, state)
	tasks, err := client.FetchTasks()
	if err != nil {
		t.Fatalf("Expected success with empty task list, got error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected empty task list, got %d tasks", len(tasks))
	}
}

// TestFetchTasksMultiple tests fetching multiple tasks (TDD - Epic 9.3)
func TestFetchTasksMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(agent.TasksResponse{
			Tasks: []agent.Task{
				{
					TaskID:     "123e4567-e89b-12d3-a456-426614174000",
					PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
					TaskType:   "backup",
					Repository: "s3:bucket/repo1",
					CreatedAt:  time.Now(),
				},
				{
					TaskID:     "223e4567-e89b-12d3-a456-426614174000",
					PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
					TaskType:   "check",
					Repository: "s3:bucket/repo2",
					CreatedAt:  time.Now(),
				},
				{
					TaskID:     "323e4567-e89b-12d3-a456-426614174000",
					PolicyID:   "434e5678-e89b-12d3-a456-426614174001",
					TaskType:   "prune",
					Repository: "s3:bucket/repo3",
					CreatedAt:  time.Now(),
				},
			},
			Count: 3,
		})
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

	client := agent.NewTaskClient(cfg, state)
	tasks, err := client.FetchTasks()
	if err != nil {
		t.Fatalf("Expected task fetch to succeed, got error: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Verify task types
	expectedTypes := []string{"backup", "check", "prune"}
	for i, task := range tasks {
		if task.TaskType != expectedTypes[i] {
			t.Errorf("Task %d: expected type %s, got %s", i, expectedTypes[i], task.TaskType)
		}
	}
}

// TestFetchTasksInvalidAgentID tests task fetch with invalid agent ID (TDD - Epic 9.3)
func TestFetchTasksInvalidAgentID(t *testing.T) {
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

	client := agent.NewTaskClient(cfg, state)
	_, err := client.FetchTasks()
	if err == nil {
		t.Fatal("Expected error for invalid agent ID")
	}
}

// TestFetchTasksServerError tests task fetch with server error (TDD - Epic 9.3)
func TestFetchTasksServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
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

	client := agent.NewTaskClient(cfg, state)
	_, err := client.FetchTasks()
	if err == nil {
		t.Fatal("Expected error for server error")
	}
}

// TestFetchTasksNetworkError tests task fetch with network error (TDD - Epic 9.3)
func TestFetchTasksNetworkError(t *testing.T) {
	cfg := &agent.Config{
		OrchestratorURL:     "http://localhost:99999", // Invalid port
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  1,
		RetryMaxAttempts:    0,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewTaskClient(cfg, state)
	_, err := client.FetchTasks()
	if err == nil {
		t.Fatal("Expected error for network error")
	}
}

// TestFetchTasksRetrySuccess tests task fetch retry mechanism (TDD - Epic 9.3)
func TestFetchTasksRetrySuccess(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(agent.TasksResponse{
			Tasks: []agent.Task{
				{
					TaskID:     "123e4567-e89b-12d3-a456-426614174000",
					PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
					TaskType:   "backup",
					Repository: "s3:bucket/repo",
					CreatedAt:  time.Now(),
				},
			},
			Count: 1,
		})
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "test-token",
		HTTPTimeoutSeconds:  30,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1,
	}

	state := &agent.State{
		AgentID:       "550e8400-e29b-41d4-a716-446655440000",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      "test-host",
	}

	client := agent.NewTaskClient(cfg, state)

	startTime := time.Now()
	tasks, err := client.FetchTasks()
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Expected task fetch to succeed after retries, got error: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
	if duration < 2*time.Second {
		t.Errorf("Expected backoff delay, but took only %v", duration)
	}
}

// TestFetchTasksInvalidTaskSchema tests handling of invalid task data (TDD - Epic 9.3)
func TestFetchTasksInvalidTaskSchema(t *testing.T) {
	testCases := []struct {
		name string
		task agent.Task
	}{
		{
			name: "Invalid taskId UUID",
			task: agent.Task{
				TaskID:     "invalid-uuid",
				PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
				TaskType:   "backup",
				Repository: "s3:bucket/repo",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "Invalid policyId UUID",
			task: agent.Task{
				TaskID:     "123e4567-e89b-12d3-a456-426614174000",
				PolicyID:   "invalid-uuid",
				TaskType:   "backup",
				Repository: "s3:bucket/repo",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "Invalid taskType",
			task: agent.Task{
				TaskID:     "123e4567-e89b-12d3-a456-426614174000",
				PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
				TaskType:   "invalid-type",
				Repository: "s3:bucket/repo",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "Empty repository",
			task: agent.Task{
				TaskID:     "123e4567-e89b-12d3-a456-426614174000",
				PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
				TaskType:   "backup",
				Repository: "",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "Missing createdAt",
			task: agent.Task{
				TaskID:     "123e4567-e89b-12d3-a456-426614174000",
				PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
				TaskType:   "backup",
				Repository: "s3:bucket/repo",
				CreatedAt:  time.Time{}, // Zero value
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(agent.TasksResponse{
					Tasks: []agent.Task{tc.task},
					Count: 1,
				})
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

			client := agent.NewTaskClient(cfg, state)
			_, err := client.FetchTasks()
			if err == nil {
				t.Fatal("Expected error for invalid task schema")
			}
		})
	}
}

// TestFetchTasksAuthorizationHeader tests that authorization header is sent (TDD - Epic 9.3)
func TestFetchTasksAuthorizationHeader(t *testing.T) {
	authHeaderReceived := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderReceived = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &agent.Config{
		OrchestratorURL:     server.URL,
		AuthenticationToken: "my-secret-token-456",
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

	client := agent.NewTaskClient(cfg, state)
	client.FetchTasks()

	expectedHeader := "Bearer my-secret-token-456"
	if authHeaderReceived != expectedHeader {
		t.Errorf("Expected Authorization header %q, got %q", expectedHeader, authHeaderReceived)
	}
}
