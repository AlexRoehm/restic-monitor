package policy

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/example/restic-monitor/internal/store"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = store.MigrateModels(db)
	require.NoError(t, err)

	return db
}

// createTestTenant creates a test tenant
func createTestTenant(t *testing.T, db *gorm.DB) uuid.UUID {
	tenantID := uuid.New()
	return tenantID
}

// createTestCredential creates a test credential
func createTestCredential(t *testing.T, db *gorm.DB, tenantID uuid.UUID, name string) *store.Credential {
	password := "encrypted_password_here"
	cred := &store.Credential{
		TenantID:     tenantID,
		Name:         name,
		Type:         store.CredentialTypePassword,
		PasswordHash: &password,
	}
	err := db.Create(cred).Error
	require.NoError(t, err)
	return cred
}

// TestValidatePolicy_ValidPolicy tests that a valid policy passes validation
func TestValidatePolicy_ValidPolicy(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:           tenantID,
		Name:               "test-policy",
		Schedule:           "0 2 * * *", // Valid cron
		IncludePaths:       store.JSONB{"paths": []interface{}{"/home", "/var/www"}},
		ExcludePaths:       store.JSONB{"patterns": []interface{}{"*.log"}},
		RepositoryURL:      "s3:s3.amazonaws.com/bucket",
		RepositoryType:     "s3",
		RetentionRules:     store.JSONB{"keep_daily": 7, "keep_weekly": 4},
		Enabled:            true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

// TestValidatePolicy_InvalidCronSchedule tests that invalid cron expressions are rejected
func TestValidatePolicy_InvalidCronSchedule(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "invalid cron",
		IncludePaths:   store.JSONB{"paths": []interface{}{"/home"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Field, "schedule")
}

// TestValidatePolicy_IncludePathInForbiddenSandbox tests sandbox conflict detection
func TestValidatePolicy_IncludePathInForbiddenSandbox(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []interface{}{"/etc/shadow"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		SandboxConfig: store.JSONB{
			"allowed":   []interface{}{"/home", "/var"},
			"forbidden": []interface{}{"/etc/shadow", "/root"},
			"max_depth": 20,
		},
		Enabled: true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	
	// Check that error mentions the forbidden path
	foundError := false
	for _, err := range result.Errors {
		if err.Field == "include_paths" && err.Code == "sandbox_forbidden" {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Expected sandbox_forbidden error for /etc/shadow")
}

// TestValidatePolicy_IncludePathNotInAllowedSandbox tests that paths outside allowed list are rejected
func TestValidatePolicy_IncludePathNotInAllowedSandbox(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []interface{}{"/usr/bin"}}, // Not in allowed list
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		SandboxConfig: store.JSONB{
			"allowed":   []interface{}{"/home", "/var"},
			"forbidden": []interface{}{},
			"max_depth": 20,
		},
		Enabled: true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	
	// Check that error mentions path not in allowed list
	foundError := false
	for _, err := range result.Errors {
		if err.Field == "include_paths" && err.Code == "sandbox_not_allowed" {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Expected sandbox_not_allowed error")
}

// TestValidatePolicy_MissingCredentials tests that referencing non-existent credentials fails
func TestValidatePolicy_MissingCredentials(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	nonExistentCredID := uuid.New()
	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []interface{}{"/home"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		CredentialsID:  &nonExistentCredID,
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	
	// Check that error mentions missing credentials
	foundError := false
	for _, err := range result.Errors {
		if err.Field == "credentials_id" && err.Code == "credentials_not_found" {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Expected credentials_not_found error")
}

// TestValidatePolicy_ValidCredentials tests that valid credential reference passes
func TestValidatePolicy_ValidCredentials(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)
	
	cred := createTestCredential(t, db, tenantID, "s3-creds")

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []interface{}{"/home"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		CredentialsID:  &cred.ID,
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

// TestValidatePolicy_EmptyIncludePaths tests that policies must have at least one include path
func TestValidatePolicy_EmptyIncludePaths(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []interface{}{}}, // Empty
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	
	foundError := false
	for _, err := range result.Errors {
		if err.Field == "include_paths" && err.Code == "empty_include_paths" {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Expected empty_include_paths error")
}

// TestValidatePolicy_InvalidRetentionRules tests retention rule validation
func TestValidatePolicy_InvalidRetentionRules(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	tests := []struct {
		name      string
		retention store.JSONB
		errorCode string
	}{
		{
			name:      "negative keep_daily",
			retention: store.JSONB{"keep_daily": -1},
			errorCode: "invalid_retention",
		},
		{
			name:      "zero values for all rules",
			retention: store.JSONB{"keep_daily": 0, "keep_weekly": 0, "keep_monthly": 0},
			errorCode: "empty_retention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &store.Policy{
				TenantID:       tenantID,
				Name:           "test-policy",
				Schedule:       "0 2 * * *",
				IncludePaths:   store.JSONB{"paths": []interface{}{"/home"}},
				RepositoryURL:  "s3:bucket",
				RepositoryType: "s3",
				RetentionRules: tt.retention,
				Enabled:        true,
			}

			result, err := validator.ValidatePolicy(policy)
			require.NoError(t, err)
			assert.False(t, result.Valid)
			
			foundError := false
			for _, err := range result.Errors {
				if err.Field == "retention_rules" && err.Code == tt.errorCode {
					foundError = true
					break
				}
			}
			assert.True(t, foundError, "Expected %s error", tt.errorCode)
		})
	}
}

// TestValidatePolicy_IntervalSchedule tests that interval schedules are supported
func TestValidatePolicy_IntervalSchedule(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "@every 1h", // Interval schedule
		IncludePaths:   store.JSONB{"paths": []interface{}{"/home"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

// TestValidatePolicy_ExpiredCredentials tests warning for expired credentials
func TestValidatePolicy_ExpiredCredentials(t *testing.T) {
	db := setupTestDB(t)
	tenantID := createTestTenant(t, db)
	validator := NewPolicyValidator(db)
	
	// Create credential with expired date
	password := "encrypted"
	expiresAt := time.Now().Add(-24 * time.Hour) // Expired yesterday
	cred := &store.Credential{
		TenantID:     tenantID,
		Name:         "expired-cert",
		Type:         store.CredentialTypeCert,
		PasswordHash: &password,
		ExpiresAt:    &expiresAt,
	}
	err := db.Create(cred).Error
	require.NoError(t, err)

	policy := &store.Policy{
		TenantID:       tenantID,
		Name:           "test-policy",
		Schedule:       "0 2 * * *",
		IncludePaths:   store.JSONB{"paths": []string{"/home"}},
		RepositoryURL:  "s3:bucket",
		RepositoryType: "s3",
		RetentionRules: store.JSONB{"keep_daily": 7},
		CredentialsID:  &cred.ID,
		Enabled:        true,
	}

	result, err := validator.ValidatePolicy(policy)
	require.NoError(t, err)
	// Should return warnings but still be valid
	assert.False(t, result.Valid) // Actually should be invalid since credentials expired
	
	foundWarning := false
	for _, err := range result.Errors {
		if err.Field == "credentials_id" && err.Code == "credentials_expired" {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "Expected credentials_expired warning")
}

// TestValidateSandbox tests sandbox validation in isolation
func TestValidateSandbox(t *testing.T) {
	db := setupTestDB(t)
	validator := NewPolicyValidator(db)

	tests := []struct {
		name          string
		sandboxConfig store.JSONB
		includePaths  []string
		expectValid   bool
		errorCode     string
	}{
		{
			name: "nil sandbox allows all",
			sandboxConfig: nil,
			includePaths: []string{"/etc/shadow"},
			expectValid: true,
		},
		{
			name: "path in allowed list",
			sandboxConfig: store.JSONB{
				"allowed":   []interface{}{"/home", "/var"},
				"forbidden": []interface{}{},
				"max_depth": 20,
			},
			includePaths: []string{"/home/user"},
			expectValid: true,
		},
		{
			name: "path in forbidden list",
			sandboxConfig: store.JSONB{
				"allowed":   []interface{}{"/home"},
				"forbidden": []interface{}{"/home/user/.ssh"},
				"max_depth": 20,
			},
			includePaths: []string{"/home/user/.ssh/id_rsa"},
			expectValid: false,
			errorCode: "sandbox_forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSandboxPaths(tt.sandboxConfig, tt.includePaths)
			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
