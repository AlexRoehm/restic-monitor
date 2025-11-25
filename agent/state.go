package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// State represents the persistent agent state stored in state.json
type State struct {
	AgentID       string    `json:"agentId"`
	RegisteredAt  time.Time `json:"registeredAt"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	Hostname      string    `json:"hostname"`
}

// LoadState loads the agent state from a JSON file.
// Returns nil,nil if the file does not exist (first run scenario).
// Returns error if the file exists but is corrupt or invalid.
func LoadState(path string) (*State, error) {
	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil // First run, no state file yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat state file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Check for empty file
	if len(data) == 0 {
		return nil, fmt.Errorf("state file is empty")
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Validate loaded state
	if err := validateState(&state); err != nil {
		return nil, fmt.Errorf("invalid state in file: %w", err)
	}

	return &state, nil
}

// SaveState saves the agent state to a JSON file with atomic write.
// Creates parent directories if they don't exist.
// Uses atomic write (write to temp file, then rename) to prevent corruption.
func SaveState(path string, state *State) error {
	// Validate state before saving
	if err := validateState(state); err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// validateState validates the state fields
func validateState(state *State) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Agent ID is required and must be a valid UUID
	if state.AgentID == "" {
		return fmt.Errorf("agentId is required")
	}
	if _, err := uuid.Parse(state.AgentID); err != nil {
		return fmt.Errorf("agentId must be a valid UUID: %w", err)
	}

	// Hostname is required
	if state.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	// RegisteredAt must be set
	if state.RegisteredAt.IsZero() {
		return fmt.Errorf("registeredAt is required")
	}

	// LastHeartbeat is optional but if set, should not be before RegisteredAt
	if !state.LastHeartbeat.IsZero() && state.LastHeartbeat.Before(state.RegisteredAt) {
		return fmt.Errorf("lastHeartbeat cannot be before registeredAt")
	}

	return nil
}

// GetDirectory returns the directory path from a file path
func GetDirectory(filePath string) string {
	return filepath.Dir(filePath)
}
