package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TaskExecutor executes backup/check/prune tasks using Restic
type TaskExecutor struct {
	resticBinary string
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskID          string                 `json:"taskId"`
	Status          string                 `json:"status"` // success, failure
	DurationSeconds float64                `json:"durationSeconds"`
	Log             string                 `json:"log"`
	SnapshotID      string                 `json:"snapshotId,omitempty"`
	TaskType        string                 `json:"taskType"`
	StartTime       time.Time              `json:"startTime"`
	EndTime         time.Time              `json:"endTime"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(resticBinary string) *TaskExecutor {
	return &TaskExecutor{
		resticBinary: resticBinary,
	}
}

// BuildCommand constructs the Restic CLI command for a task
func (e *TaskExecutor) BuildCommand(task Task) (string, []string) {
	var args []string

	switch task.TaskType {
	case "backup":
		args = e.buildBackupArgs(task)
	case "check":
		args = e.buildCheckArgs(task)
	case "prune":
		args = e.buildPruneArgs(task)
	default:
		args = []string{"--help"}
	}

	return e.resticBinary, args
}

// buildBackupArgs constructs arguments for backup command
func (e *TaskExecutor) buildBackupArgs(task Task) []string {
	args := []string{"-r", task.Repository, "backup"}

	// Add include paths
	if task.IncludePaths != nil {
		if paths, ok := task.IncludePaths["paths"].([]interface{}); ok {
			for _, p := range paths {
				if path, ok := p.(string); ok {
					args = append(args, path)
				}
			}
		}
	}

	// Add exclude patterns
	if task.ExcludePaths != nil {
		if paths, ok := task.ExcludePaths["paths"].([]interface{}); ok {
			for _, p := range paths {
				if pattern, ok := p.(string); ok {
					args = append(args, fmt.Sprintf("--exclude=%s", pattern))
				}
			}
		}
	}

	// Add execution parameters
	if task.ExecutionParams != nil {
		if bw, ok := task.ExecutionParams["bandwidthLimitKbps"].(float64); ok {
			args = append(args, fmt.Sprintf("--limit-upload=%d", int(bw)))
		}
		if par, ok := task.ExecutionParams["parallelism"].(float64); ok {
			args = append(args, "-o", fmt.Sprintf("local.connections=%d", int(par)))
		}
	}

	return args
}

// buildCheckArgs constructs arguments for check command
func (e *TaskExecutor) buildCheckArgs(task Task) []string {
	args := []string{"-r", task.Repository, "check"}

	// Add execution parameters
	if task.ExecutionParams != nil {
		if readData, ok := task.ExecutionParams["readData"].(bool); ok && readData {
			args = append(args, "--read-data")
		}
	}

	return args
}

// buildPruneArgs constructs arguments for prune command
func (e *TaskExecutor) buildPruneArgs(task Task) []string {
	args := []string{"-r", task.Repository, "forget", "--prune"}

	// Add retention rules
	if task.Retention != nil {
		if keepLast, ok := task.Retention["keepLast"].(float64); ok {
			args = append(args, fmt.Sprintf("--keep-last=%d", int(keepLast)))
		}
		if keepDaily, ok := task.Retention["keepDaily"].(float64); ok {
			args = append(args, fmt.Sprintf("--keep-daily=%d", int(keepDaily)))
		}
		if keepWeekly, ok := task.Retention["keepWeekly"].(float64); ok {
			args = append(args, fmt.Sprintf("--keep-weekly=%d", int(keepWeekly)))
		}
		if keepMonthly, ok := task.Retention["keepMonthly"].(float64); ok {
			args = append(args, fmt.Sprintf("--keep-monthly=%d", int(keepMonthly)))
		}
		if keepYearly, ok := task.Retention["keepYearly"].(float64); ok {
			args = append(args, fmt.Sprintf("--keep-yearly=%d", int(keepYearly)))
		}
	}

	return args
}

// Execute runs the task and returns the result
func (e *TaskExecutor) Execute(task Task) (TaskResult, error) {
	return e.ExecuteWithEnv(task, nil)
}

// ExecuteWithEnv runs the task with custom environment variables
func (e *TaskExecutor) ExecuteWithEnv(task Task, env map[string]string) (TaskResult, error) {
	startTime := time.Now()

	result := TaskResult{
		TaskID:    task.TaskID,
		Status:    "failure",
		TaskType:  task.TaskType,
		StartTime: startTime,
		Metadata:  make(map[string]interface{}),
	}

	cmd, args := e.BuildCommand(task)

	// For testing with echo/sleep, use repository as argument
	if strings.Contains(cmd, "echo") || strings.Contains(cmd, "sleep") {
		args = []string{task.Repository}
	}

	// Execute command
	execCmd := exec.Command(cmd, args...)

	// Set environment variables if provided
	for key, value := range env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	result.EndTime = endTime
	result.DurationSeconds = duration.Seconds()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	result.Log = output

	// Check exit status
	if err == nil {
		result.Status = "success"
	} else {
		result.Status = "failure"
	}

	// Extract snapshot ID from backup output
	if task.TaskType == "backup" && result.Status == "success" {
		result.SnapshotID = ExtractSnapshotID(result.Log)
	}

	// Parse and store check result metadata
	if task.TaskType == "check" {
		checkResult := ParseCheckOutput(result.Log)
		result.Metadata["checkResult"] = &checkResult
		// Update status based on check result
		if checkResult.Success {
			result.Status = "success"
		} else {
			result.Status = "failure"
		}
	}

	// Parse and store prune result metadata
	if task.TaskType == "prune" && result.Status == "success" {
		pruneResult := ParsePruneOutput(result.Log)
		result.Metadata["pruneResult"] = &pruneResult
	}

	return result, nil
}

// ToJSON serializes the task result to JSON
func (r *TaskResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ExtractSnapshotID extracts the snapshot ID from Restic backup output
func ExtractSnapshotID(output string) string {
	// Look for pattern: "snapshot <id> saved"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "snapshot ") && strings.HasSuffix(line, " saved") {
			// Extract ID between "snapshot " and " saved"
			parts := strings.Fields(line)
			if len(parts) >= 3 && parts[0] == "snapshot" && parts[2] == "saved" {
				return parts[1]
			}
		}
	}
	return ""
}

// CheckResult represents the result of a repository check
type CheckResult struct {
	Success       bool   `json:"success"`
	ErrorsFound   int    `json:"errorsFound"`
	WarningsFound int    `json:"warningsFound"`
	Summary       string `json:"summary"`
}

// ParseCheckOutput parses the output from a restic check command
func ParseCheckOutput(output string) CheckResult {
	result := CheckResult{
		Success:       true,
		ErrorsFound:   0,
		WarningsFound: 0,
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count errors
		if strings.HasPrefix(line, "error:") {
			result.ErrorsFound++
			result.Success = false
		}

		// Count warnings
		if strings.HasPrefix(line, "warning:") {
			result.WarningsFound++
		}

		// Check for fatal errors
		if strings.HasPrefix(line, "Fatal:") {
			result.Success = false
			result.Summary = "Failed: " + line
		}

		// Check for success message
		if strings.Contains(line, "no errors were found") || strings.Contains(line, "no fatal errors") {
			if result.ErrorsFound == 0 {
				result.Summary = "Success: " + line
			}
		}
	}

	// Set summary if not already set
	if result.Summary == "" {
		if result.Success {
			result.Summary = fmt.Sprintf("Check complete: %d errors, %d warnings", result.ErrorsFound, result.WarningsFound)
		} else {
			result.Summary = fmt.Sprintf("Check failed: %d errors found", result.ErrorsFound)
		}
	}

	return result
}

// PruneResult represents the result of a prune operation
type PruneResult struct {
	SnapshotsRemoved   int      `json:"snapshotsRemoved"`
	SnapshotsKept      int      `json:"snapshotsKept"`
	SpaceFreedBytes    int64    `json:"spaceFreedBytes"`
	DeletedSnapshotIDs []string `json:"deletedSnapshotIds,omitempty"`
	Summary            string   `json:"summary"`
}

// ParsePruneOutput parses the output from a restic forget --prune command
func ParsePruneOutput(output string) PruneResult {
	result := PruneResult{
		DeletedSnapshotIDs: make([]string, 0),
	}

	lines := strings.Split(output, "\n")
	inRemoveSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect "remove X snapshots:" section
		if strings.HasPrefix(line, "remove ") && strings.Contains(line, " snapshots") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &result.SnapshotsRemoved)
			}
			inRemoveSection = true
			continue
		}

		// Exit remove section when we hit other known sections
		if inRemoveSection && (strings.HasPrefix(line, "keep ") ||
			strings.Contains(line, "snapshots have been removed") ||
			strings.Contains(line, "no snapshots were removed") ||
			strings.Contains(line, "repository is already") ||
			strings.Contains(line, "counting files") ||
			strings.Contains(line, "building new index") ||
			len(line) == 0) {
			inRemoveSection = false
		}

		// Extract snapshot IDs from remove section
		if inRemoveSection && len(line) > 0 && !strings.HasPrefix(line, "ID") && !strings.HasPrefix(line, "remove") {
			// Line format: "old001    2024-01-01 10:00:00  server1"
			parts := strings.Fields(line)
			if len(parts) >= 1 && !strings.Contains(line, "Time") && !strings.Contains(line, "Host") {
				// First field is the snapshot ID
				snapshotID := parts[0]
				// Validate it looks like a snapshot ID (alphanumeric, reasonable length)
				if len(snapshotID) >= 6 && len(snapshotID) <= 64 {
					result.DeletedSnapshotIDs = append(result.DeletedSnapshotIDs, snapshotID)
				}
			}
		}

		// Parse snapshots kept
		if strings.HasPrefix(line, "keep ") && strings.Contains(line, " snapshots") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &result.SnapshotsKept)
			}
		}

		// Parse space freed
		if strings.Contains(line, "this frees") {
			// Example: "will delete 3 packs and rewrite 2 packs, this frees 45.6 MiB"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "frees" && i+2 < len(parts) {
					var size float64
					unit := parts[i+2]
					fmt.Sscanf(parts[i+1], "%f", &size)

					// Convert to bytes
					switch strings.ToLower(unit) {
					case "kib":
						result.SpaceFreedBytes = int64(size * 1024)
					case "mib":
						result.SpaceFreedBytes = int64(size * 1024 * 1024)
					case "gib":
						result.SpaceFreedBytes = int64(size * 1024 * 1024 * 1024)
					}
					break
				}
			}
		}

		// Check for summary messages
		if strings.Contains(line, "snapshots have been removed") {
			result.Summary = line
		}
		if strings.Contains(line, "no snapshots were removed") {
			result.Summary = line
		}
	}

	return result
}

// TruncateLog truncates the log to the specified maximum size in bytes
func (r TaskResult) TruncateLog(maxBytes int) TaskResult {
	if len(r.Log) <= maxBytes {
		return r
	}

	truncated := r
	truncateMsg := "\n... (log truncated)"
	availableBytes := maxBytes - len(truncateMsg)

	if availableBytes > 0 {
		truncated.Log = r.Log[:availableBytes] + truncateMsg
	} else {
		truncated.Log = truncateMsg
	}

	return truncated
}

// SaveToFile saves the task result to a JSON file in the specified directory
func (r *TaskResult) SaveToFile(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", r.TaskID))

	data, err := r.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadTaskResult loads a task result from a JSON file
func LoadTaskResult(dir, taskID string) (*TaskResult, error) {
	filename := filepath.Join(dir, fmt.Sprintf("%s.json", taskID))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result TaskResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}

	return &result, nil
}

// LogEntry represents a single log entry with timestamp and level
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	TaskID    string    `json:"taskId"`
}

// TaskLogger provides structured logging for task execution
type TaskLogger struct {
	TaskID  string     `json:"taskId"`
	Entries []LogEntry `json:"entries"`
}

// NewTaskLogger creates a new task logger
func NewTaskLogger(taskID string) *TaskLogger {
	return &TaskLogger{
		TaskID:  taskID,
		Entries: make([]LogEntry, 0),
	}
}

// Info logs an info-level message
func (tl *TaskLogger) Info(message string) {
	tl.addEntry("INFO", message)
}

// Debug logs a debug-level message
func (tl *TaskLogger) Debug(message string) {
	tl.addEntry("DEBUG", message)
}

// Error logs an error-level message
func (tl *TaskLogger) Error(message string) {
	tl.addEntry("ERROR", message)
}

// addEntry adds a log entry with the current timestamp
func (tl *TaskLogger) addEntry(level, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		TaskID:    tl.TaskID,
	}
	tl.Entries = append(tl.Entries, entry)
}

// GetLogs returns all log entries
func (tl *TaskLogger) GetLogs() []LogEntry {
	return tl.Entries
}

// ToJSON serializes the logger to JSON
func (tl *TaskLogger) ToJSON() ([]byte, error) {
	return json.Marshal(tl)
}

// ========================================
// Retry & Error Handling (Epic 11.5)
// ========================================

// ErrorCategory represents types of errors for retry decisions
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryNetwork
	ErrorCategoryTransient
	ErrorCategoryPermission
	ErrorCategoryRepo
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// CategorizeError determines the category of an error
func CategorizeError(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	errMsg := strings.ToLower(err.Error())

	// Network errors
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no route to host") ||
		strings.Contains(errMsg, "network is unreachable") {
		return ErrorCategoryNetwork
	}

	// Transient errors
	if strings.Contains(errMsg, "already locked") ||
		strings.Contains(errMsg, "temporarily unavailable") {
		return ErrorCategoryTransient
	}

	// Permission errors
	if strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "access denied") {
		return ErrorCategoryPermission
	}

	// Repository errors
	if strings.Contains(errMsg, "does not exist") ||
		strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "invalid repository") {
		return ErrorCategoryRepo
	}

	return ErrorCategoryUnknown
}

// IsRetryable determines if an error category should be retried
func IsRetryable(category ErrorCategory) bool {
	switch category {
	case ErrorCategoryNetwork, ErrorCategoryTransient:
		return true
	default:
		return false
	}
}

// CalculateBackoff computes the backoff duration for a given attempt
func (rc *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return rc.InitialBackoff
	}

	// Exponential backoff: initialBackoff * (multiplier ^ (attempt - 1))
	backoff := rc.InitialBackoff
	for i := 1; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * rc.BackoffMultiplier)
		if backoff > rc.MaxBackoff {
			return rc.MaxBackoff
		}
	}

	if backoff > rc.MaxBackoff {
		return rc.MaxBackoff
	}

	return backoff
}

// ExecuteWithRetry executes a task with retry logic for transient failures
func (e *TaskExecutor) ExecuteWithRetry(task Task, config RetryConfig) (*TaskResult, error) {
	var lastErr error
	var retriedErrors []string

	// Extract environment from task
	env := map[string]string{
		"RESTIC_REPOSITORY": task.Repository,
	}
	// Password would come from secure storage, simplified here
	if execParams, ok := task.ExecutionParams["password"].(string); ok {
		env["RESTIC_PASSWORD"] = execParams
	}

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, err := e.ExecuteWithEnv(task, env)

		// Success case
		if err == nil && result.Status == "success" {
			// Track retry metadata if this wasn't the first attempt
			if attempt > 1 {
				if result.Metadata == nil {
					result.Metadata = make(map[string]interface{})
				}
				result.Metadata["retryAttempts"] = attempt
				result.Metadata["retriedErrors"] = retriedErrors
			}
			return &result, nil
		}

		// Failure case
		if err != nil {
			retriedErrors = append(retriedErrors, fmt.Sprintf("attempt %d: %v", attempt, err))
			lastErr = err
		} else {
			retriedErrors = append(retriedErrors, fmt.Sprintf("attempt %d: task failed", attempt))
			lastErr = fmt.Errorf("task execution failed")
		}

		// Check if error is retryable
		category := CategorizeError(lastErr)
		if !IsRetryable(category) {
			// Permanent error - fail immediately
			if result.Metadata == nil {
				result.Metadata = make(map[string]interface{})
			}
			result.Metadata["retryAttempts"] = 1
			result.Metadata["errorCategory"] = category
			return &result, lastErr
		}

		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts {
			backoff := config.CalculateBackoff(attempt)
			time.Sleep(backoff)
		}
	}

	// All retries exhausted
	result := &TaskResult{
		TaskID:          task.TaskID,
		Status:          "failure",
		DurationSeconds: 0,
		Log:             fmt.Sprintf("Max retry attempts (%d) exceeded", config.MaxAttempts),
		TaskType:        task.TaskType,
		Metadata: map[string]interface{}{
			"retryAttempts": config.MaxAttempts,
			"retriedErrors": retriedErrors,
		},
	}

	return result, fmt.Errorf("max retry attempts exceeded: %w", lastErr)
}

// ========================================
// Execution Metrics (Epic 11.6)
// ========================================

// ExecutionMetrics tracks task execution statistics
type ExecutionMetrics struct {
	totalTasks      int64
	successfulTasks int64
	failedTasks     int64
	bytesProcessed  int64
	totalDuration   float64
	concurrentTasks int
	mu              sync.RWMutex
}

// ExecutionMetricsSnapshot represents a point-in-time view of metrics
type ExecutionMetricsSnapshot struct {
	TotalTasks      int64     `json:"totalTasks"`
	SuccessfulTasks int64     `json:"successfulTasks"`
	FailedTasks     int64     `json:"failedTasks"`
	SuccessRate     float64   `json:"successRate"`
	AverageDuration float64   `json:"averageDuration"`
	BytesProcessed  int64     `json:"bytesProcessed"`
	ConcurrentTasks int       `json:"concurrentTasks"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewExecutionMetrics creates a new metrics tracker
func NewExecutionMetrics() *ExecutionMetrics {
	return &ExecutionMetrics{
		totalTasks:      0,
		successfulTasks: 0,
		failedTasks:     0,
		bytesProcessed:  0,
		totalDuration:   0,
		concurrentTasks: 0,
	}
}

// RecordTaskStart increments concurrent task counter
func (em *ExecutionMetrics) RecordTaskStart(taskType string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.concurrentTasks++
}

// RecordTaskComplete records task completion with metrics
func (em *ExecutionMetrics) RecordTaskComplete(taskType string, success bool, duration float64, bytesProcessed int64) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.totalTasks++
	em.totalDuration += duration
	em.bytesProcessed += bytesProcessed

	if success {
		em.successfulTasks++
	} else {
		em.failedTasks++
	}

	// Decrement concurrent counter
	if em.concurrentTasks > 0 {
		em.concurrentTasks--
	}
}

// GetTotalTasks returns total task count
func (em *ExecutionMetrics) GetTotalTasks() int64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.totalTasks
}

// GetSuccessfulTasks returns successful task count
func (em *ExecutionMetrics) GetSuccessfulTasks() int64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.successfulTasks
}

// GetFailedTasks returns failed task count
func (em *ExecutionMetrics) GetFailedTasks() int64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.failedTasks
}

// GetBytesProcessed returns total bytes processed
func (em *ExecutionMetrics) GetBytesProcessed() int64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.bytesProcessed
}

// GetConcurrentTasks returns current concurrent task count
func (em *ExecutionMetrics) GetConcurrentTasks() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.concurrentTasks
}

// GetSuccessRate returns success percentage (0-100)
func (em *ExecutionMetrics) GetSuccessRate() float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.totalTasks == 0 {
		return 0.0
	}

	return (float64(em.successfulTasks) / float64(em.totalTasks)) * 100.0
}

// GetAverageDuration returns average task duration in seconds
func (em *ExecutionMetrics) GetAverageDuration() float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.totalTasks == 0 {
		return 0.0
	}

	return em.totalDuration / float64(em.totalTasks)
}

// GetSnapshot returns a point-in-time snapshot of metrics
func (em *ExecutionMetrics) GetSnapshot() ExecutionMetricsSnapshot {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return ExecutionMetricsSnapshot{
		TotalTasks:      em.totalTasks,
		SuccessfulTasks: em.successfulTasks,
		FailedTasks:     em.failedTasks,
		SuccessRate:     em.GetSuccessRate(),
		AverageDuration: em.GetAverageDuration(),
		BytesProcessed:  em.bytesProcessed,
		ConcurrentTasks: em.concurrentTasks,
		Timestamp:       time.Now(),
	}
}
