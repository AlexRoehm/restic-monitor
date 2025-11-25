package agent

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration loaded from agent.yaml
type Config struct {
	OrchestratorURL          string `yaml:"orchestratorUrl"`
	AgentID                  string `yaml:"agentId"`
	AuthenticationToken      string `yaml:"authenticationToken"`
	HostnameOverride         string `yaml:"hostnameOverride"`
	LogLevel                 string `yaml:"logLevel"`
	LogFile                  string `yaml:"logFile"`
	PollingIntervalSeconds   int    `yaml:"pollingIntervalSeconds"`
	HeartbeatIntervalSeconds int    `yaml:"heartbeatIntervalSeconds"`
	MaxConcurrentJobs        int    `yaml:"maxConcurrentJobs"`
	HTTPTimeoutSeconds       int    `yaml:"httpTimeoutSeconds"`
	RetryMaxAttempts         int    `yaml:"retryMaxAttempts"`
	RetryBackoffSeconds      int    `yaml:"retryBackoffSeconds"`
	StateFile                string `yaml:"stateFile"`
	TempDir                  string `yaml:"tempDir"`
}

// LoadConfig loads and validates the agent configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Apply environment variable overrides
	applyEnvironmentOverrides(&cfg)

	// Validate
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults sets default values for optional configuration fields
// Note: Only applies defaults if field was not explicitly set in YAML
// Zero values from YAML will be validated as-is
func applyDefaults(cfg *Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	// For integer fields, we can't distinguish between "not set" and "set to zero"
	// in standard YAML unmarshaling. We'll validate first, then apply defaults
	// only if validation would pass with the default value.

	// These fields have default values that will be applied during validation
	// if they're unset (0). The validation will fail if explicitly set to invalid values.
	if cfg.StateFile == "" {
		cfg.StateFile = "/var/lib/restic-agent/state.json"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp/restic-agent"
	}
}

// applyEnvironmentOverrides applies environment variable overrides to the configuration
func applyEnvironmentOverrides(cfg *Config) {
	if val := os.Getenv("RESTIC_AGENT_ORCHESTRATOR_URL"); val != "" {
		cfg.OrchestratorURL = val
	}
	if val := os.Getenv("RESTIC_AGENT_ID"); val != "" {
		cfg.AgentID = val
	}
	if val := os.Getenv("RESTIC_AGENT_AUTH_TOKEN"); val != "" {
		cfg.AuthenticationToken = val
	}
	if val := os.Getenv("RESTIC_AGENT_HOSTNAME"); val != "" {
		cfg.HostnameOverride = val
	}
	if val := os.Getenv("RESTIC_AGENT_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}
	if val := os.Getenv("RESTIC_AGENT_LOG_FILE"); val != "" {
		cfg.LogFile = val
	}
	if val := os.Getenv("RESTIC_AGENT_POLLING_INTERVAL"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.PollingIntervalSeconds = intVal
		}
	}
	if val := os.Getenv("RESTIC_AGENT_HEARTBEAT_INTERVAL"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.HeartbeatIntervalSeconds = intVal
		}
	}
	if val := os.Getenv("RESTIC_AGENT_MAX_CONCURRENT_JOBS"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.MaxConcurrentJobs = intVal
		}
	}
	if val := os.Getenv("RESTIC_AGENT_HTTP_TIMEOUT"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.HTTPTimeoutSeconds = intVal
		}
	}
	if val := os.Getenv("RESTIC_AGENT_STATE_FILE"); val != "" {
		cfg.StateFile = val
	}
	if val := os.Getenv("RESTIC_AGENT_TEMP_DIR"); val != "" {
		cfg.TempDir = val
	}
}

// validateConfig validates the configuration and returns detailed error messages
func validateConfig(cfg *Config) error {
	var errors []string

	// Required fields
	if cfg.OrchestratorURL == "" {
		errors = append(errors, "orchestratorUrl is required")
	} else {
		if err := validateOrchestratorURL(cfg.OrchestratorURL); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if cfg.AuthenticationToken == "" {
		errors = append(errors, "authenticationToken is required")
	}

	// Log level validation
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if cfg.LogLevel != "" {
		valid := false
		for _, level := range validLogLevels {
			if cfg.LogLevel == level {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("logLevel must be one of: %s", strings.Join(validLogLevels, ", ")))
		}
	}

	// Apply defaults for unset numeric fields and validate
	if cfg.PollingIntervalSeconds == 0 {
		cfg.PollingIntervalSeconds = 30
	}
	if cfg.PollingIntervalSeconds < 5 || cfg.PollingIntervalSeconds > 3600 {
		errors = append(errors, "pollingIntervalSeconds must be between 5 and 3600")
	}

	if cfg.HeartbeatIntervalSeconds == 0 {
		cfg.HeartbeatIntervalSeconds = 60
	}
	if cfg.HeartbeatIntervalSeconds < 10 || cfg.HeartbeatIntervalSeconds > 3600 {
		errors = append(errors, "heartbeatIntervalSeconds must be between 10 and 3600")
	}

	if cfg.MaxConcurrentJobs == 0 {
		cfg.MaxConcurrentJobs = 2
	}
	if cfg.MaxConcurrentJobs < 1 || cfg.MaxConcurrentJobs > 10 {
		errors = append(errors, "maxConcurrentJobs must be between 1 and 10")
	}

	if cfg.HTTPTimeoutSeconds == 0 {
		cfg.HTTPTimeoutSeconds = 30
	}
	if cfg.HTTPTimeoutSeconds < 5 || cfg.HTTPTimeoutSeconds > 300 {
		errors = append(errors, "httpTimeoutSeconds must be between 5 and 300")
	}

	if cfg.RetryMaxAttempts == 0 {
		cfg.RetryMaxAttempts = 3
	}
	if cfg.RetryMaxAttempts < 0 || cfg.RetryMaxAttempts > 10 {
		errors = append(errors, "retryMaxAttempts must be between 0 and 10")
	}

	if cfg.RetryBackoffSeconds == 0 {
		cfg.RetryBackoffSeconds = 5
	}
	if cfg.RetryBackoffSeconds < 1 || cfg.RetryBackoffSeconds > 60 {
		errors = append(errors, "retryBackoffSeconds must be between 1 and 60")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateOrchestratorURL validates the orchestrator URL format
func validateOrchestratorURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("orchestratorUrl cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("orchestratorUrl is not a valid URL: %w", err)
	}

	// Must be http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("orchestratorUrl must use http or https protocol")
	}

	// Check host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("orchestratorUrl must include a host")
	}

	// No trailing slash
	if strings.HasSuffix(urlStr, "/") {
		return fmt.Errorf("orchestratorUrl must not end with trailing slash")
	}

	return nil
}
