package api

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Policy name validation constants
const (
	policyNameMinLength = 3
	policyNameMaxLength = 100
)

var (
	// policyNamePattern allows alphanumeric, hyphens, and underscores
	policyNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// cronPattern validates standard 5-field cron expressions
	cronPattern = regexp.MustCompile(`^(@(annually|yearly|monthly|weekly|daily|hourly))|((((\*|([0-9]|[1-5][0-9]))|(\*|([0-9]|[1-5][0-9]))/([0-9]|[1-5][0-9])|(([0-9]|[1-5][0-9])-([0-9]|[1-5][0-9]))|(([0-9]|[1-5][0-9])(,([0-9]|[1-5][0-9]))+))\s+((\*|([0-9]|1[0-9]|2[0-3]))|(\*|([0-9]|1[0-9]|2[0-3]))/([0-9]|1[0-9]|2[0-3])|(([0-9]|1[0-9]|2[0-3])-([0-9]|1[0-9]|2[0-3]))|(([0-9]|1[0-9]|2[0-3])(,([0-9]|1[0-9]|2[0-3]))+))\s+((\*|([1-9]|[12][0-9]|3[01]))|(\*|([1-9]|[12][0-9]|3[01]))/([1-9]|[12][0-9]|3[01])|(([1-9]|[12][0-9]|3[01])-([1-9]|[12][0-9]|3[01]))|(([1-9]|[12][0-9]|3[01])(,([1-9]|[12][0-9]|3[01]))+))\s+((\*|([1-9]|1[012]))|(\*|([1-9]|1[012]))/([1-9]|1[012])|(([1-9]|1[012])-([1-9]|1[012]))|(([1-9]|1[012])(,([1-9]|1[012]))+))\s+((\*|[0-7])|(\*|[0-7])/[0-7]|([0-7]-[0-7])|([0-7](,[0-7])+))))$`)

	// s3BucketPattern validates S3 bucket names
	s3BucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)

	// keepWithinPattern validates duration format like "30d", "12h", "2w"
	keepWithinPattern = regexp.MustCompile(`^[0-9]+[hdwmy]$`)
)

// validatePolicyName validates policy name according to schema rules
func validatePolicyName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}

	if len(name) < policyNameMinLength {
		return fmt.Errorf("name must be at least %d characters", policyNameMinLength)
	}

	if len(name) > policyNameMaxLength {
		return fmt.Errorf("name must not exceed %d characters", policyNameMaxLength)
	}

	if !policyNamePattern.MatchString(name) {
		return fmt.Errorf("name can only contain alphanumeric characters, hyphens, and underscores")
	}

	// Check start and end characters
	firstChar := rune(name[0])
	lastChar := rune(name[len(name)-1])

	if !isAlphanumeric(firstChar) {
		return fmt.Errorf("name must start with an alphanumeric character")
	}

	if !isAlphanumeric(lastChar) {
		return fmt.Errorf("name must end with an alphanumeric character")
	}

	return nil
}

// validateCronSchedule validates cron expression format
func validateCronSchedule(schedule string) error {
	if schedule == "" {
		return fmt.Errorf("schedule is required")
	}

	// Simple validation: check for 5 fields
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return fmt.Errorf("invalid cron expression: must have 5 fields (minute hour day month weekday)")
	}

	// Validate each field
	minute := fields[0]
	hour := fields[1]
	day := fields[2]
	month := fields[3]
	weekday := fields[4]

	// Validate minute (0-59)
	if err := validateCronField(minute, 0, 59, "minute"); err != nil {
		return err
	}

	// Validate hour (0-23)
	if err := validateCronField(hour, 0, 23, "hour"); err != nil {
		return err
	}

	// Validate day (1-31)
	if err := validateCronField(day, 1, 31, "day"); err != nil {
		return err
	}

	// Validate month (1-12)
	if err := validateCronField(month, 1, 12, "month"); err != nil {
		return err
	}

	// Validate weekday (0-7, where 0 and 7 are Sunday)
	if err := validateCronField(weekday, 0, 7, "weekday"); err != nil {
		return err
	}

	return nil
}

// validateCronField validates a single cron field
func validateCronField(field string, min, max int, fieldName string) error {
	// Allow wildcards
	if field == "*" || field == "?" {
		return nil
	}

	// Handle step values (*/5)
	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid %s: invalid step format", fieldName)
		}
		// Validate step is numeric
		var step int
		if _, err := fmt.Sscanf(parts[1], "%d", &step); err != nil {
			return fmt.Errorf("invalid %s: step must be numeric", fieldName)
		}
		if step <= 0 {
			return fmt.Errorf("invalid %s: step must be positive", fieldName)
		}
		return nil
	}

	// Handle ranges (5-10)
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid %s: invalid range format", fieldName)
		}
		var start, end int
		if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
			return fmt.Errorf("invalid %s: range start must be numeric", fieldName)
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
			return fmt.Errorf("invalid %s: range end must be numeric", fieldName)
		}
		if start < min || start > max {
			return fmt.Errorf("invalid %s: range start %d out of bounds [%d-%d]", fieldName, start, min, max)
		}
		if end < min || end > max {
			return fmt.Errorf("invalid %s: range end %d out of bounds [%d-%d]", fieldName, end, min, max)
		}
		return nil
	}

	// Handle lists (1,5,10)
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			var val int
			if _, err := fmt.Sscanf(part, "%d", &val); err != nil {
				return fmt.Errorf("invalid %s: list value must be numeric", fieldName)
			}
			if val < min || val > max {
				return fmt.Errorf("invalid %s: value %d out of bounds [%d-%d]", fieldName, val, min, max)
			}
		}
		return nil
	}

	// Single numeric value
	var val int
	if _, err := fmt.Sscanf(field, "%d", &val); err != nil {
		return fmt.Errorf("invalid cron expression: %s must be numeric, range, list, or *", fieldName)
	}
	if val < min || val > max {
		return fmt.Errorf("invalid %s: value %d out of bounds [%d-%d]", fieldName, val, min, max)
	}

	return nil
}

// validateIncludePaths validates include paths array
func validateIncludePaths(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("at least one include path is required")
	}

	if len(paths) > 100 {
		return fmt.Errorf("cannot exceed 100 include paths")
	}

	for _, path := range paths {
		if path == "" {
			return fmt.Errorf("include path cannot be empty")
		}

		if len(path) > 4096 {
			return fmt.Errorf("include path cannot exceed 4096 characters")
		}

		// Check if path is absolute (Unix or Windows)
		if !filepath.IsAbs(path) && !isWindowsAbsolute(path) {
			return fmt.Errorf("include path must be absolute (got: %s)", path)
		}
	}

	return nil
}

// validateExcludePaths validates exclude paths/patterns array
func validateExcludePaths(paths []string) error {
	// Exclude paths are optional
	if len(paths) == 0 {
		return nil
	}

	if len(paths) > 1000 {
		return fmt.Errorf("cannot exceed 1000 exclude paths")
	}

	for _, path := range paths {
		if path == "" {
			return fmt.Errorf("exclude path cannot be empty")
		}

		if len(path) > 4096 {
			return fmt.Errorf("exclude path cannot exceed 4096 characters")
		}
	}

	return nil
}

// validateRepositoryType validates repository type
func validateRepositoryType(repoType string) error {
	if repoType == "" {
		return fmt.Errorf("repository type is required")
	}

	validTypes := map[string]bool{
		"s3":          true,
		"rest-server": true,
		"fs":          true,
		"sftp":        true,
	}

	if !validTypes[repoType] {
		return fmt.Errorf("unsupported repository type: %s (supported: s3, rest-server, fs, sftp)", repoType)
	}

	return nil
}

// validateS3Repository validates S3 repository configuration
func validateS3Repository(config map[string]interface{}) error {
	bucket, ok := config["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("S3 bucket is required")
	}

	// Validate bucket name format
	if len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("S3 bucket name must be 3-63 characters")
	}

	if !s3BucketPattern.MatchString(bucket) {
		return fmt.Errorf("invalid S3 bucket name: must be lowercase alphanumeric with hyphens and dots")
	}

	return nil
}

// validateRestServerRepository validates REST server repository configuration
func validateRestServerRepository(config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("REST server URL is required")
	}

	// Validate URL has http or https scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("REST server URL must use http or https scheme")
	}

	return nil
}

// validateFilesystemRepository validates filesystem repository configuration
func validateFilesystemRepository(config map[string]interface{}) error {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("filesystem path is required")
	}

	// Validate path is absolute (Unix or Windows)
	if !filepath.IsAbs(path) && !isWindowsAbsolute(path) {
		return fmt.Errorf("filesystem path must be absolute (got: %s)", path)
	}

	return nil
}

// validateSFTPRepository validates SFTP repository configuration
func validateSFTPRepository(config map[string]interface{}) error {
	host, ok := config["host"].(string)
	if !ok || host == "" {
		return fmt.Errorf("SFTP host is required")
	}

	user, ok := config["user"].(string)
	if !ok || user == "" {
		return fmt.Errorf("SFTP user is required")
	}

	path, ok := config["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("SFTP path is required")
	}

	// Validate port if provided
	if portVal, ok := config["port"]; ok {
		var port int
		switch v := portVal.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		default:
			return fmt.Errorf("SFTP port must be an integer")
		}

		if port < 1 || port > 65535 {
			return fmt.Errorf("SFTP port must be between 1 and 65535 (got: %d)", port)
		}
	}

	return nil
}

// validateRetentionRules validates retention rules
func validateRetentionRules(rules map[string]interface{}) error {
	if len(rules) == 0 {
		return fmt.Errorf("at least one retention rule is required")
	}

	// Validate each rule
	intRules := []string{"keepLast", "keepHourly", "keepDaily", "keepWeekly", "keepMonthly", "keepYearly"}
	for _, ruleName := range intRules {
		if val, ok := rules[ruleName]; ok {
			var intVal int
			switch v := val.(type) {
			case int:
				intVal = v
			case float64:
				intVal = int(v)
			default:
				return fmt.Errorf("%s must be an integer", ruleName)
			}

			if intVal <= 0 {
				return fmt.Errorf("%s must be positive (got: %d)", ruleName, intVal)
			}
		}
	}

	// Validate keepWithin if present
	if val, ok := rules["keepWithin"]; ok {
		keepWithin, ok := val.(string)
		if !ok {
			return fmt.Errorf("keepWithin must be a string")
		}

		if !keepWithinPattern.MatchString(keepWithin) {
			return fmt.Errorf("invalid keepWithin format: must be like '30d', '12h', '2w' (got: %s)", keepWithin)
		}
	}

	return nil
}

// validateBandwidthLimit validates bandwidth limit
func validateBandwidthLimit(limit *int) error {
	if limit == nil {
		return nil // Optional field
	}

	if *limit <= 0 {
		return fmt.Errorf("bandwidth limit must be positive (got: %d KB/s)", *limit)
	}

	if *limit > 1000000 {
		return fmt.Errorf("bandwidth limit cannot exceed 1000000 KB/s (1 GB/s, got: %d KB/s)", *limit)
	}

	return nil
}

// validateParallelFiles validates parallel files setting
func validateParallelFiles(files *int) error {
	if files == nil {
		return nil // Optional field
	}

	if *files < 1 || *files > 32 {
		return fmt.Errorf("parallel files must be between 1 and 32 (got: %d)", *files)
	}

	return nil
}

// validatePolicyRequest validates a complete policy request
func validatePolicyRequest(req *PolicyRequest) error {
	// Validate name
	if err := validatePolicyName(req.Name); err != nil {
		return err
	}

	// Validate schedule
	if err := validateCronSchedule(req.Schedule); err != nil {
		return err
	}

	// Validate include paths
	if err := validateIncludePaths(req.IncludePaths); err != nil {
		return err
	}

	// Validate exclude paths
	if err := validateExcludePaths(req.ExcludePaths); err != nil {
		return err
	}

	// Validate repository
	if req.Repository == nil || len(req.Repository) == 0 {
		return fmt.Errorf("repository is required")
	}

	repoType, ok := req.Repository["type"].(string)
	if !ok {
		return fmt.Errorf("repository type is required")
	}

	if err := validateRepositoryType(repoType); err != nil {
		return err
	}

	// Validate repository type-specific fields
	switch repoType {
	case "s3":
		if err := validateS3Repository(req.Repository); err != nil {
			return err
		}
	case "rest-server":
		if err := validateRestServerRepository(req.Repository); err != nil {
			return err
		}
	case "fs":
		if err := validateFilesystemRepository(req.Repository); err != nil {
			return err
		}
	case "sftp":
		if err := validateSFTPRepository(req.Repository); err != nil {
			return err
		}
	}

	// Validate retention rules
	if err := validateRetentionRules(req.RetentionRules); err != nil {
		return err
	}

	// Validate bandwidth limit
	if err := validateBandwidthLimit(req.BandwidthLimitKBps); err != nil {
		return err
	}

	// Validate parallel files
	if err := validateParallelFiles(req.ParallelFiles); err != nil {
		return err
	}

	return nil
}

// isAlphanumeric checks if a rune is alphanumeric
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isWindowsAbsolute checks if a path is a Windows absolute path
func isWindowsAbsolute(path string) bool {
	if len(path) < 3 {
		return false
	}
	// Check for drive letter pattern (C:\, D:\, etc.)
	if (path[0] >= 'A' && path[0] <= 'Z' || path[0] >= 'a' && path[0] <= 'z') && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	return false
}
