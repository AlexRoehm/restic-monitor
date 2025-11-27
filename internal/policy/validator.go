package policy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/example/restic-monitor/internal/store"
)

// PolicyValidator validates backup policies for conflicts and errors
type PolicyValidator struct {
	db *gorm.DB
}

// NewPolicyValidator creates a new policy validator
func NewPolicyValidator(db *gorm.DB) *PolicyValidator {
	return &PolicyValidator{db: db}
}

// ValidationResult contains the result of policy validation
type ValidationResult struct {
	Valid  bool                    `json:"valid"`
	Errors []store.ValidationError `json:"errors,omitempty"`
}

// ValidatePolicy performs comprehensive validation on a policy
func (v *PolicyValidator) ValidatePolicy(policy *store.Policy) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []store.ValidationError{},
	}

	// Validate schedule
	if err := v.validateSchedule(policy.Schedule); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, store.ValidationError{
			Field:   "schedule",
			Message: err.Error(),
			Code:    "invalid_schedule",
		})
	}

	// Validate include paths
	if errs := v.validateIncludePaths(policy); len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, errs...)
	}

	// Validate sandbox configuration
	if policy.SandboxConfig != nil {
		if errs := v.validateSandbox(policy); len(errs) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, errs...)
		}
	}

	// Validate retention rules
	if errs := v.validateRetentionRules(policy.RetentionRules); len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, errs...)
	}

	// Validate credentials if specified
	if policy.CredentialsID != nil {
		if errs := v.validateCredentials(policy); len(errs) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, errs...)
		}
	}

	// Validate hooks if specified (Phase 5 - deferred for now)
	// TODO: Implement hook validation when Epic 14 is complete

	return result, nil
}

// validateSchedule validates cron or interval schedule syntax
func (v *PolicyValidator) validateSchedule(schedule string) error {
	// Check if it's a special schedule (@hourly, @daily, @every, etc.)
	if strings.HasPrefix(schedule, "@") {
		// Special schedules are valid if recognized by cron parser
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(schedule); err != nil {
			return fmt.Errorf("invalid schedule '%s': %w", schedule, err)
		}
		return nil
	}

	// Validate standard cron expression (5 fields)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(schedule); err != nil {
		return fmt.Errorf("invalid cron schedule '%s': %w", schedule, err)
	}

	return nil
}

// validateIncludePaths validates that include paths are not empty and conform to sandbox
func (v *PolicyValidator) validateIncludePaths(policy *store.Policy) []store.ValidationError {
	errors := []store.ValidationError{}

	// Extract paths from JSONB
	paths, ok := policy.IncludePaths["paths"]
	if !ok {
		errors = append(errors, store.ValidationError{
			Field:   "include_paths",
			Message: "include_paths must contain a 'paths' array",
			Code:    "missing_paths_field",
		})
		return errors
	}

	// Convert to string slice
	pathSlice, ok := paths.([]interface{})
	if !ok {
		errors = append(errors, store.ValidationError{
			Field:   "include_paths",
			Message: "include_paths.paths must be an array",
			Code:    "invalid_paths_type",
		})
		return errors
	}

	// Check that at least one path is specified
	if len(pathSlice) == 0 {
		errors = append(errors, store.ValidationError{
			Field:   "include_paths",
			Message: "at least one include path is required",
			Code:    "empty_include_paths",
		})
		return errors
	}

	// Validate each path against sandbox if configured
	if policy.SandboxConfig != nil {
		stringPaths := make([]string, len(pathSlice))
		for i, p := range pathSlice {
			if strPath, ok := p.(string); ok {
				stringPaths[i] = strPath
			}
		}
		if sandboxErrs := v.validateSandboxPaths(policy.SandboxConfig, stringPaths); len(sandboxErrs) > 0 {
			errors = append(errors, sandboxErrs...)
		}
	}

	return errors
}

// validateSandbox validates sandbox configuration
func (v *PolicyValidator) validateSandbox(policy *store.Policy) []store.ValidationError {
	errors := []store.ValidationError{}

	// Parse sandbox config
	allowed, _ := policy.SandboxConfig["allowed"].([]interface{})
	forbidden, _ := policy.SandboxConfig["forbidden"].([]interface{})

	// Check for conflicts: paths that are both allowed and forbidden
	allowedSet := make(map[string]bool)
	for _, a := range allowed {
		if str, ok := a.(string); ok {
			allowedSet[str] = true
		}
	}

	for _, f := range forbidden {
		if str, ok := f.(string); ok {
			if allowedSet[str] {
				errors = append(errors, store.ValidationError{
					Field:   "sandbox_config",
					Message: fmt.Sprintf("path '%s' cannot be both allowed and forbidden", str),
					Code:    "sandbox_conflict",
				})
			}
		}
	}

	return errors
}

// ValidateSandboxPaths validates that paths conform to sandbox restrictions
// Returns error for backward compatibility (used by external callers)
func (v *PolicyValidator) ValidateSandboxPaths(sandboxConfig store.JSONB, paths []string) error {
	errs := v.validateSandboxPaths(sandboxConfig, paths)
	if len(errs) > 0 {
		return fmt.Errorf("%s", errs[0].Message)
	}
	return nil
}

// validateSandboxPaths validates paths and returns detailed errors
func (v *PolicyValidator) validateSandboxPaths(sandboxConfig store.JSONB, paths []string) []store.ValidationError {
	if sandboxConfig == nil {
		return nil // No sandbox = unrestricted
	}

	errors := []store.ValidationError{}

	// Extract allowed and forbidden lists
	allowedRaw, _ := sandboxConfig["allowed"].([]interface{})
	forbiddenRaw, _ := sandboxConfig["forbidden"].([]interface{})

	allowed := make([]string, 0, len(allowedRaw))
	for _, a := range allowedRaw {
		if str, ok := a.(string); ok {
			allowed = append(allowed, filepath.Clean(str))
		}
	}

	forbidden := make([]string, 0, len(forbiddenRaw))
	for _, f := range forbiddenRaw {
		if str, ok := f.(string); ok {
			forbidden = append(forbidden, filepath.Clean(str))
		}
	}

	// Validate each path
	for _, path := range paths {
		cleanPath := filepath.Clean(path)

		// Check forbidden list first
		for _, forbiddenPath := range forbidden {
			if strings.HasPrefix(cleanPath, forbiddenPath) {
				errors = append(errors, store.ValidationError{
					Field:   "include_paths",
					Message: fmt.Sprintf("path '%s' is forbidden by sandbox (matches '%s')", path, forbiddenPath),
					Code:    "sandbox_forbidden",
				})
				continue // Skip to next path
			}
		}

		// If allowed list is specified, path must be under one of the allowed roots
		if len(allowed) > 0 {
			isAllowed := false
			for _, allowedPath := range allowed {
				if strings.HasPrefix(cleanPath, allowedPath) || cleanPath == allowedPath {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				errors = append(errors, store.ValidationError{
					Field:   "include_paths",
					Message: fmt.Sprintf("path '%s' is not in allowed sandbox paths", path),
					Code:    "sandbox_not_allowed",
				})
			}
		}
	}

	return errors
}

// validateRetentionRules validates retention policy rules
func (v *PolicyValidator) validateRetentionRules(rules store.JSONB) []store.ValidationError {
	errors := []store.ValidationError{}

	if len(rules) == 0 {
		errors = append(errors, store.ValidationError{
			Field:   "retention_rules",
			Message: "retention rules are required",
			Code:    "missing_retention",
		})
		return errors
	}

	// Check for at least one retention rule > 0
	hasRule := false
	validKeys := []string{"keep_last", "keep_daily", "keep_weekly", "keep_monthly", "keep_yearly"}

	for key, value := range rules {
		// Check if key is valid
		isValidKey := false
		for _, validKey := range validKeys {
			if key == validKey {
				isValidKey = true
				break
			}
		}

		if !isValidKey {
			continue // Skip unknown keys
		}

		// Convert value to number
		var numValue float64
		switch v := value.(type) {
		case float64:
			numValue = v
		case int:
			numValue = float64(v)
		default:
			errors = append(errors, store.ValidationError{
				Field:   "retention_rules",
				Message: fmt.Sprintf("retention rule '%s' must be a number", key),
				Code:    "invalid_retention",
			})
			continue
		}

		// Check for negative values
		if numValue < 0 {
			errors = append(errors, store.ValidationError{
				Field:   "retention_rules",
				Message: fmt.Sprintf("retention rule '%s' cannot be negative", key),
				Code:    "invalid_retention",
			})
		}

		if numValue > 0 {
			hasRule = true
		}
	}

	if !hasRule {
		errors = append(errors, store.ValidationError{
			Field:   "retention_rules",
			Message: "at least one retention rule must be greater than 0",
			Code:    "empty_retention",
		})
	}

	return errors
}

// validateCredentials validates that referenced credentials exist and are valid
func (v *PolicyValidator) validateCredentials(policy *store.Policy) []store.ValidationError {
	errors := []store.ValidationError{}

	if policy.CredentialsID == nil {
		return errors
	}

	// Check if credential exists
	var cred store.Credential
	if err := v.db.Where("id = ? AND tenant_id = ?", policy.CredentialsID, policy.TenantID).First(&cred).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			errors = append(errors, store.ValidationError{
				Field:   "credentials_id",
				Message: "referenced credentials do not exist",
				Code:    "credentials_not_found",
			})
		} else {
			errors = append(errors, store.ValidationError{
				Field:   "credentials_id",
				Message: fmt.Sprintf("error checking credentials: %v", err),
				Code:    "credentials_error",
			})
		}
		return errors
	}

	// Check if credentials are expired
	if cred.ExpiresAt != nil && cred.ExpiresAt.Before(store.NowFunc()) {
		errors = append(errors, store.ValidationError{
			Field:   "credentials_id",
			Message: fmt.Sprintf("credentials expired on %s", cred.ExpiresAt.Format("2006-01-02")),
			Code:    "credentials_expired",
		})
	}

	return errors
}
