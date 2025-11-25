package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/example/restic-monitor/agent"
)

// DiagnosticResult represents the result of a diagnostic check
type DiagnosticResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass, fail, warn
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// DiagnosticReport contains all diagnostic results
type DiagnosticReport struct {
	Timestamp time.Time          `json:"timestamp"`
	Overall   string             `json:"overall"` // pass, fail
	Checks    []DiagnosticResult `json:"checks"`
}

func runDiagnostics(configPath string) int {
	report := DiagnosticReport{
		Timestamp: time.Now().UTC(),
		Checks:    []DiagnosticResult{},
	}

	// Check 1: Configuration file exists and is readable
	configResult := checkConfiguration(configPath)
	report.Checks = append(report.Checks, configResult)

	var cfg *agent.Config
	if configResult.Status == "pass" {
		// Load configuration for subsequent checks
		var err error
		cfg, err = agent.LoadConfig(configPath)
		if err != nil {
			report.Checks = append(report.Checks, DiagnosticResult{
				Name:    "Configuration Validation",
				Status:  "fail",
				Message: "Configuration file is invalid",
				Details: err.Error(),
			})
		} else {
			report.Checks = append(report.Checks, DiagnosticResult{
				Name:    "Configuration Validation",
				Status:  "pass",
				Message: "Configuration is valid",
			})
		}
	}

	// Check 2: State file accessibility
	if cfg != nil {
		stateResult := checkStateFile(cfg.StateFile)
		report.Checks = append(report.Checks, stateResult)
	}

	// Check 3: Orchestrator connectivity
	if cfg != nil {
		connectivityResult := checkOrchestratorConnectivity(cfg)
		report.Checks = append(report.Checks, connectivityResult)
	}

	// Check 4: File permissions
	if cfg != nil {
		permResult := checkPermissions(configPath, cfg)
		report.Checks = append(report.Checks, permResult)
	}

	// Check 5: Disk space for temp directory
	if cfg != nil {
		diskResult := checkDiskSpace(cfg.TempDir)
		report.Checks = append(report.Checks, diskResult)
	}

	// Determine overall status
	report.Overall = "pass"
	for _, check := range report.Checks {
		if check.Status == "fail" {
			report.Overall = "fail"
			break
		}
	}

	// Print report
	printReport(report)

	// Return exit code
	if report.Overall == "fail" {
		return 1
	}
	return 0
}

func checkConfiguration(configPath string) DiagnosticResult {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return DiagnosticResult{
			Name:    "Configuration File",
			Status:  "fail",
			Message: "Configuration file does not exist",
			Details: fmt.Sprintf("File not found: %s", configPath),
		}
	}
	if err != nil {
		return DiagnosticResult{
			Name:    "Configuration File",
			Status:  "fail",
			Message: "Cannot access configuration file",
			Details: err.Error(),
		}
	}

	// Check if readable
	_, err = os.ReadFile(configPath)
	if err != nil {
		return DiagnosticResult{
			Name:    "Configuration File",
			Status:  "fail",
			Message: "Cannot read configuration file",
			Details: err.Error(),
		}
	}

	return DiagnosticResult{
		Name:    "Configuration File",
		Status:  "pass",
		Message: fmt.Sprintf("Configuration file exists and is readable: %s", configPath),
	}
}

func checkStateFile(stateFilePath string) DiagnosticResult {
	dir := agent.GetDirectory(stateFilePath)

	// Check if directory exists or can be created
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// Try to create it
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return DiagnosticResult{
				Name:    "State File Directory",
				Status:  "fail",
				Message: "Cannot create state file directory",
				Details: err.Error(),
			}
		}
		return DiagnosticResult{
			Name:    "State File Directory",
			Status:  "pass",
			Message: "State file directory can be created",
			Details: fmt.Sprintf("Directory: %s", dir),
		}
	}
	if err != nil {
		return DiagnosticResult{
			Name:    "State File Directory",
			Status:  "fail",
			Message: "Cannot access state file directory",
			Details: err.Error(),
		}
	}

	// Check if directory is writable
	if !info.IsDir() {
		return DiagnosticResult{
			Name:    "State File Directory",
			Status:  "fail",
			Message: "State file path is not a directory",
			Details: fmt.Sprintf("Path: %s", dir),
		}
	}

	// Try to write a test file
	testFile := dir + "/.test"
	err = os.WriteFile(testFile, []byte("test"), 0600)
	if err != nil {
		return DiagnosticResult{
			Name:    "State File Directory",
			Status:  "fail",
			Message: "State file directory is not writable",
			Details: err.Error(),
		}
	}
	os.Remove(testFile)

	return DiagnosticResult{
		Name:    "State File Directory",
		Status:  "pass",
		Message: "State file directory is writable",
		Details: fmt.Sprintf("Directory: %s", dir),
	}
}

func checkOrchestratorConnectivity(cfg *agent.Config) DiagnosticResult {
	// Create HTTP client with timeout
	timeout := time.Duration(cfg.HTTPTimeoutSeconds) * time.Second
	if cfg.HTTPTimeoutSeconds == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
	}

	// Try to connect to orchestrator
	url := cfg.OrchestratorURL + "/health"
	resp, err := client.Get(url)
	if err != nil {
		return DiagnosticResult{
			Name:    "Orchestrator Connectivity",
			Status:  "fail",
			Message: "Cannot connect to orchestrator",
			Details: fmt.Sprintf("URL: %s, Error: %v", cfg.OrchestratorURL, err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return DiagnosticResult{
			Name:    "Orchestrator Connectivity",
			Status:  "pass",
			Message: "Successfully connected to orchestrator",
			Details: fmt.Sprintf("URL: %s, Status: %d", cfg.OrchestratorURL, resp.StatusCode),
		}
	}

	return DiagnosticResult{
		Name:    "Orchestrator Connectivity",
		Status:  "warn",
		Message: "Orchestrator responded with non-2xx status",
		Details: fmt.Sprintf("URL: %s, Status: %d", cfg.OrchestratorURL, resp.StatusCode),
	}
}

func checkPermissions(configPath string, cfg *agent.Config) DiagnosticResult {
	// Check configuration file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		return DiagnosticResult{
			Name:    "File Permissions",
			Status:  "fail",
			Message: "Cannot check file permissions",
			Details: err.Error(),
		}
	}

	mode := info.Mode().Perm()

	// Warn if config file is world-readable (contains sensitive token)
	if mode&0004 != 0 {
		return DiagnosticResult{
			Name:    "File Permissions",
			Status:  "warn",
			Message: "Configuration file is world-readable (contains sensitive data)",
			Details: fmt.Sprintf("Current permissions: %o (recommended: 0600)", mode),
		}
	}

	return DiagnosticResult{
		Name:    "File Permissions",
		Status:  "pass",
		Message: "File permissions are secure",
		Details: fmt.Sprintf("Configuration file: %o", mode),
	}
}

func checkDiskSpace(tempDir string) DiagnosticResult {
	// Create temp directory if it doesn't exist
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return DiagnosticResult{
			Name:    "Disk Space",
			Status:  "fail",
			Message: "Cannot create temp directory",
			Details: err.Error(),
		}
	}

	// Note: Full disk space checking would require platform-specific code
	// For now, just verify the directory is accessible
	return DiagnosticResult{
		Name:    "Disk Space",
		Status:  "pass",
		Message: "Temp directory is accessible",
		Details: fmt.Sprintf("Directory: %s", tempDir),
	}
}

func printReport(report DiagnosticReport) {
	// Print as JSON for machine parsing
	if os.Getenv("DIAGNOSTIC_FORMAT") == "json" {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Print human-readable format
	fmt.Println("=== Agent Diagnostics Report ===")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Overall Status: %s\n\n", getStatusSymbol(report.Overall))

	for i, check := range report.Checks {
		fmt.Printf("%d. %s: %s\n", i+1, check.Name, getStatusSymbol(check.Status))
		fmt.Printf("   %s\n", check.Message)
		if check.Details != "" {
			fmt.Printf("   Details: %s\n", check.Details)
		}
		fmt.Println()
	}

	// Summary
	passCount := 0
	warnCount := 0
	failCount := 0
	for _, check := range report.Checks {
		switch check.Status {
		case "pass":
			passCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
		}
	}

	fmt.Printf("Summary: %d passed, %d warnings, %d failed\n", passCount, warnCount, failCount)
}

func getStatusSymbol(status string) string {
	switch status {
	case "pass":
		return "✓ PASS"
	case "warn":
		return "⚠ WARN"
	case "fail":
		return "✗ FAIL"
	default:
		return status
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	exitCode := runDiagnostics(configPath)
	os.Exit(exitCode)
}
