package agent_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/example/restic-monitor/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempConfig creates a temporary agent.yaml file for testing
func writeTempConfig(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "agent.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)
	return tmpFile
}

// TestLoadConfigMinimal tests loading minimal valid configuration (TDD - Epic 8.1)
func TestLoadConfigMinimal(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "test-token-123"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)

	// Verify required fields
	assert.Equal(t, "https://backup.example.com", cfg.OrchestratorURL)
	assert.Equal(t, "test-token-123", cfg.AuthenticationToken)

	// Verify defaults were applied
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 30, cfg.PollingIntervalSeconds)
	assert.Equal(t, 60, cfg.HeartbeatIntervalSeconds)
	assert.Equal(t, 2, cfg.MaxConcurrentJobs)
}

// TestLoadConfigFull tests loading full configuration with all fields (TDD - Epic 8.1)
func TestLoadConfigFull(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.mycompany.com:8080"
agentId: "550e8400-e29b-41d4-a716-446655440000"
authenticationToken: "secure-token-here"
hostnameOverride: "prod-web-01"
logLevel: "debug"
logFile: "/var/log/restic-agent/agent.log"
pollingIntervalSeconds: 15
heartbeatIntervalSeconds: 30
maxConcurrentJobs: 4
httpTimeoutSeconds: 60
retryMaxAttempts: 5
retryBackoffSeconds: 10
stateFile: "/var/lib/restic-agent/state.json"
tempDir: "/var/tmp/restic-agent"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "https://backup.mycompany.com:8080", cfg.OrchestratorURL)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", cfg.AgentID)
	assert.Equal(t, "secure-token-here", cfg.AuthenticationToken)
	assert.Equal(t, "prod-web-01", cfg.HostnameOverride)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/var/log/restic-agent/agent.log", cfg.LogFile)
	assert.Equal(t, 15, cfg.PollingIntervalSeconds)
	assert.Equal(t, 30, cfg.HeartbeatIntervalSeconds)
	assert.Equal(t, 4, cfg.MaxConcurrentJobs)
	assert.Equal(t, 60, cfg.HTTPTimeoutSeconds)
	assert.Equal(t, 5, cfg.RetryMaxAttempts)
	assert.Equal(t, 10, cfg.RetryBackoffSeconds)
	assert.Equal(t, "/var/lib/restic-agent/state.json", cfg.StateFile)
	assert.Equal(t, "/var/tmp/restic-agent", cfg.TempDir)
}

// TestLoadConfigMissingOrchestratorURL tests that missing orchestratorUrl fails validation (TDD - Epic 8.1)
func TestLoadConfigMissingOrchestratorURL(t *testing.T) {
	configYAML := `
authenticationToken: "token"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	_, err := agent.LoadConfig(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "orchestratorUrl")
	assert.Contains(t, err.Error(), "required")
}

// TestLoadConfigMissingAuthToken tests that missing authenticationToken fails validation (TDD - Epic 8.1)
func TestLoadConfigMissingAuthToken(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	_, err := agent.LoadConfig(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authenticationToken")
	assert.Contains(t, err.Error(), "required")
}

// TestLoadConfigInvalidYAML tests handling of malformed YAML (TDD - Epic 8.1)
func TestLoadConfigInvalidYAML(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: [this is invalid yaml
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	_, err := agent.LoadConfig(tmpFile)
	assert.Error(t, err)
}

// TestLoadConfigFileNotFound tests handling of missing config file (TDD - Epic 8.1)
func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := agent.LoadConfig("/nonexistent/agent.yaml")
	assert.Error(t, err)
}

// TestConfigValidationPollingInterval tests pollingIntervalSeconds range validation (TDD - Epic 8.1)
func TestConfigValidationPollingInterval(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		shouldError bool
	}{
		{"Too low", 4, true},
		{"Min valid", 5, false},
		{"Normal", 30, false},
		{"Max valid", 3600, false},
		{"Too high", 3601, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
pollingIntervalSeconds: ` + strconv.Itoa(tt.value) + `
`
			tmpFile := writeTempConfig(t, configYAML)
			defer os.Remove(tmpFile)

			_, err := agent.LoadConfig(tmpFile)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "pollingIntervalSeconds")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigDefaultPollingInterval tests that pollingIntervalSeconds gets a default value when not set (TDD - Epic 8.1)
func TestConfigDefaultPollingInterval(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.PollingIntervalSeconds, "should apply default polling interval")
}

// TestConfigValidationInvalidLogLevel tests log level validation (TDD - Epic 8.1)
func TestConfigValidationInvalidLogLevel(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
logLevel: "verbose"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	_, err := agent.LoadConfig(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logLevel")
	assert.Contains(t, err.Error(), "debug, info, warn, error")
}

// TestConfigEnvironmentOverride tests environment variable override (TDD - Epic 8.1)
func TestConfigEnvironmentOverride(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "file-token"
logLevel: "info"
pollingIntervalSeconds: 30
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	// Set environment variables
	os.Setenv("RESTIC_AGENT_HOSTNAME", "test-host")
	os.Setenv("RESTIC_AGENT_POLLING_INTERVAL", "15")
	os.Setenv("RESTIC_AGENT_LOG_LEVEL", "debug")
	defer os.Unsetenv("RESTIC_AGENT_HOSTNAME")
	defer os.Unsetenv("RESTIC_AGENT_POLLING_INTERVAL")
	defer os.Unsetenv("RESTIC_AGENT_LOG_LEVEL")

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)

	// Environment variables should override file values
	assert.Equal(t, "test-host", cfg.HostnameOverride)
	assert.Equal(t, 15, cfg.PollingIntervalSeconds)
	assert.Equal(t, "debug", cfg.LogLevel)

	// File value should remain for non-overridden field
	assert.Equal(t, "file-token", cfg.AuthenticationToken)
}

// TestConfigEnvironmentAuthToken tests auth token from environment (TDD - Epic 8.1)
func TestConfigEnvironmentAuthToken(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	// Set auth token via environment variable
	os.Setenv("RESTIC_AGENT_AUTH_TOKEN", "env-secret-token")
	defer os.Unsetenv("RESTIC_AGENT_AUTH_TOKEN")

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "env-secret-token", cfg.AuthenticationToken)
}

// TestConfigValidationInvalidURL tests URL format validation (TDD - Epic 8.1)
func TestConfigValidationInvalidURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"Valid HTTPS", "https://backup.example.com", false},
		{"Valid HTTP", "http://backup.example.com", false},
		{"Valid with port", "https://backup.example.com:8443", false},
		{"Invalid protocol", "ftp://backup.example.com", true},
		{"No protocol", "backup.example.com", true},
		{"Trailing slash", "https://backup.example.com/", true},
		{"Empty URL", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `
orchestratorUrl: "` + tt.url + `"
authenticationToken: "token"
`
			tmpFile := writeTempConfig(t, configYAML)
			defer os.Remove(tmpFile)

			_, err := agent.LoadConfig(tmpFile)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigValidationMaxConcurrentJobs tests maxConcurrentJobs range validation (TDD - Epic 8.1)
func TestConfigValidationMaxConcurrentJobs(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		shouldError bool
	}{
		{"Min valid", 1, false},
		{"Normal", 2, false},
		{"Max valid", 10, false},
		{"Too high", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
maxConcurrentJobs: ` + strconv.Itoa(tt.value) + `
`
			tmpFile := writeTempConfig(t, configYAML)
			defer os.Remove(tmpFile)

			_, err := agent.LoadConfig(tmpFile)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "maxConcurrentJobs")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigDefaultMaxConcurrentJobs tests that maxConcurrentJobs gets a default value when not set (TDD - Epic 8.1)
func TestConfigDefaultMaxConcurrentJobs(t *testing.T) {
	configYAML := `
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
`
	tmpFile := writeTempConfig(t, configYAML)
	defer os.Remove(tmpFile)

	cfg, err := agent.LoadConfig(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.MaxConcurrentJobs, "should apply default max concurrent jobs")
}
