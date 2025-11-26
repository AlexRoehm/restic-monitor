package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/restic-monitor/agent"
)

const AgentVersion = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Load configuration
	cfg, err := agent.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting restic-monitor agent v%s", AgentVersion)
	log.Printf("Orchestrator URL: %s", cfg.OrchestratorURL)
	log.Printf("Polling interval: %ds", cfg.PollingIntervalSeconds)
	log.Printf("Heartbeat interval: %ds", cfg.HeartbeatIntervalSeconds)
	log.Printf("State file: %s", cfg.StateFile)

	// Register with orchestrator (handles first-time registration)
	log.Printf("Checking agent registration status...")
	if err := agent.Register(cfg); err != nil {
		log.Fatalf("Failed to register with orchestrator: %v", err)
	}

	// Load state (should now have AgentID after registration)
	state, err := agent.LoadState(cfg.StateFile)
	if err != nil {
		log.Fatalf("Failed to load state: %v", err)
	}
	if state == nil || state.AgentID == "" {
		log.Fatalf("Registration completed but state is invalid")
	}
	log.Printf("Agent registered: AgentID=%s, Hostname=%s", state.AgentID, state.Hostname)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Create the polling loop
	pollingLoop := agent.NewPollingLoop(cfg, state, AgentVersion)

	// Create executor with concurrency control (EPIC 15)
	executor := agent.NewTaskExecutorWithConcurrency("restic", &cfg.Concurrency)
	pollingLoop.SetExecutor(executor)

	log.Printf("Agent initialized, starting polling loop...")

	// Start the agent (blocks until context is cancelled)
	if err := pollingLoop.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Agent error: %v", err)
	}

	// Save final state
	if err := agent.SaveState(cfg.StateFile, state); err != nil {
		log.Printf("Warning: Failed to save final state: %v", err)
	}

	log.Printf("Agent shutdown complete")
}
