package store

import "time"

// EPIC 16 - Structured types for JSONB fields

// NowFunc returns the current time (can be overridden for testing)
var NowFunc = time.Now

// SandboxConfig defines path access restrictions for an agent
type SandboxConfig struct {
	Allowed   []string `json:"allowed"`   // Whitelist of allowed paths (e.g., ["/home", "/var/www"])
	Forbidden []string `json:"forbidden"` // Blacklist of forbidden paths (e.g., ["/etc/shadow", "/root"])
	MaxDepth  int      `json:"max_depth"` // Maximum directory traversal depth (e.g., 20)
}

// HookDefinition represents a hook to execute before or after a backup
type HookDefinition struct {
	ID         string                 `json:"id"`          // Hook template ID from Epic 14
	Name       string                 `json:"name"`        // Hook name for display
	Parameters map[string]interface{} `json:"parameters"`  // Hook-specific parameters
	Timeout    int                    `json:"timeout"`     // Timeout in seconds
}

// ValidationError represents a policy validation error
type ValidationError struct {
	Field   string `json:"field"`   // Field that failed validation
	Message string `json:"message"` // Human-readable error message
	Code    string `json:"code"`    // Error code for programmatic handling
}

// CredentialType constants
const (
	CredentialTypePassword = "password" // Repository password
	CredentialTypeCert     = "cert"     // TLS certificate + key
	CredentialTypeAWS      = "aws"      // AWS S3 credentials
	CredentialTypeGCS      = "gcs"      // Google Cloud Storage credentials
)

// ValidationStatus constants
const (
	ValidationStatusValid   = "valid"   // Policy passes all validation checks
	ValidationStatusInvalid = "invalid" // Policy has errors that prevent execution
	ValidationStatusWarning = "warning" // Policy has warnings but can execute
)
