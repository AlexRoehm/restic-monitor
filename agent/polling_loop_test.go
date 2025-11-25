package agent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
)

// TestNewPollingLoop tests polling loop creation (TDD - Epic 9)
func TestNewPollingLoop(t *testing.T) {
	cfg := &agent.Config{
		AgentID:                "123e4567-e89b-12d3-a456-426614174000",
		AuthenticationToken:    "test-token",
		OrchestratorURL:        "http://localhost:8080",
		PollingIntervalSeconds: 30,
		RetryMaxAttempts:       3,
		RetryBackoffSeconds:    1,
		HTTPTimeoutSeconds:     10,
	}

	state := &agent.State{
		AgentID:  "123e4567-e89b-12d3-a456-426614174000",
		Hostname: "test-host",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")

	if loop == nil {
		t.Fatal("Expected polling loop, got nil")
	}

	if loop.GetQueueSize() != 0 {
		t.Errorf("Expected empty queue, got size %d", loop.GetQueueSize())
	}
}

// TestPollingLoopSingleIteration tests one loop iteration (TDD - Epic 9)
func TestPollingLoopSingleIteration(t *testing.T) {
	// Create mock server
	heartbeatCalls := 0
	taskCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/heartbeat" {
			heartbeatCalls++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"heartbeat received"}`))
			return
		}

		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/tasks" {
			taskCalls++
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &agent.Config{
		AgentID:                "123e4567-e89b-12d3-a456-426614174000",
		AuthenticationToken:    "test-token",
		OrchestratorURL:        server.URL,
		PollingIntervalSeconds: 1,
		RetryMaxAttempts:       3,
		RetryBackoffSeconds:    1,
		HTTPTimeoutSeconds:     5,
	}

	state := &agent.State{
		AgentID:  "123e4567-e89b-12d3-a456-426614174000",
		Hostname: "test-host",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")
	loop.SetLogPrefix("[TEST]")

	// Start loop with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run in goroutine
	done := make(chan error)
	go func() {
		done <- loop.Start(ctx)
	}()

	// Wait for context timeout
	err := <-done
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Verify at least one iteration occurred
	metrics := loop.GetMetrics()
	if metrics.LoopCount < 1 {
		t.Errorf("Expected at least 1 loop iteration, got %d", metrics.LoopCount)
	}

	if heartbeatCalls < 1 {
		t.Errorf("Expected at least 1 heartbeat call, got %d", heartbeatCalls)
	}

	if taskCalls < 1 {
		t.Errorf("Expected at least 1 task call, got %d", taskCalls)
	}
}

// TestPollingLoopWithTasks tests task queuing (TDD - Epic 9)
func TestPollingLoopWithTasks(t *testing.T) {
	// Create mock server that returns tasks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/heartbeat" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"heartbeat received"}`))
			return
		}

		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/tasks" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"tasks": [
					{
						"taskId": "223e4567-e89b-12d3-a456-426614174001",
						"policyId": "323e4567-e89b-12d3-a456-426614174002",
						"taskType": "backup",
						"repository": "s3:bucket/repo",
						"createdAt": "2025-01-01T00:00:00Z"
					}
				],
				"count": 1
			}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &agent.Config{
		AgentID:                "123e4567-e89b-12d3-a456-426614174000",
		AuthenticationToken:    "test-token",
		OrchestratorURL:        server.URL,
		PollingIntervalSeconds: 1,
		RetryMaxAttempts:       3,
		RetryBackoffSeconds:    1,
		HTTPTimeoutSeconds:     5,
	}

	state := &agent.State{
		AgentID:  "123e4567-e89b-12d3-a456-426614174000",
		Hostname: "test-host",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")
	loop.SetLogPrefix("[TEST]")

	// Start loop with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Run in goroutine
	done := make(chan error)
	go func() {
		done <- loop.Start(ctx)
	}()

	// Wait for context timeout
	<-done

	// Verify tasks were queued
	metrics := loop.GetMetrics()
	if metrics.TotalTasksFetched < 1 {
		t.Errorf("Expected at least 1 task fetched, got %d", metrics.TotalTasksFetched)
	}

	queueSize := loop.GetQueueSize()
	if queueSize != 1 {
		t.Errorf("Expected queue size 1, got %d", queueSize)
	}
}

// TestPollingLoopStop tests graceful shutdown (TDD - Epic 9)
func TestPollingLoopStop(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/heartbeat" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","message":"heartbeat received"}`))
			return
		}

		if r.URL.Path == "/agents/123e4567-e89b-12d3-a456-426614174000/tasks" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &agent.Config{
		AgentID:                "123e4567-e89b-12d3-a456-426614174000",
		AuthenticationToken:    "test-token",
		OrchestratorURL:        server.URL,
		PollingIntervalSeconds: 1,
		RetryMaxAttempts:       3,
		RetryBackoffSeconds:    1,
		HTTPTimeoutSeconds:     5,
	}

	state := &agent.State{
		AgentID:  "123e4567-e89b-12d3-a456-426614174000",
		Hostname: "test-host",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")
	loop.SetLogPrefix("[TEST]")

	// Start loop
	ctx := context.Background()
	done := make(chan error)
	go func() {
		done <- loop.Start(ctx)
	}()

	// Let it run for a bit
	time.Sleep(500 * time.Millisecond)

	// Stop the loop
	go loop.Stop()

	// Should stop cleanly
	err := <-done
	if err != nil {
		t.Errorf("Expected nil error on stop, got %v", err)
	}
}

// TestPollingLoopFormatMetrics tests metrics formatting (TDD - Epic 9)
func TestPollingLoopFormatMetrics(t *testing.T) {
	cfg := &agent.Config{
		AgentID:                "123e4567-e89b-12d3-a456-426614174000",
		AuthenticationToken:    "test-token",
		OrchestratorURL:        "http://localhost:8080",
		PollingIntervalSeconds: 30,
		RetryMaxAttempts:       3,
		RetryBackoffSeconds:    1,
		HTTPTimeoutSeconds:     10,
	}

	state := &agent.State{
		AgentID:  "123e4567-e89b-12d3-a456-426614174000",
		Hostname: "test-host",
	}

	loop := agent.NewPollingLoop(cfg, state, "1.0.0")

	formatted := loop.FormatMetrics()
	if formatted == "" {
		t.Error("Expected non-empty formatted metrics")
	}

	// Should contain key metrics
	if !strings.Contains(formatted, "Loops:") {
		t.Error("Expected formatted metrics to contain 'Loops:'")
	}
	if !strings.Contains(formatted, "Tasks:") {
		t.Error("Expected formatted metrics to contain 'Tasks:'")
	}
	if !strings.Contains(formatted, "Heartbeats:") {
		t.Error("Expected formatted metrics to contain 'Heartbeats:'")
	}
}
