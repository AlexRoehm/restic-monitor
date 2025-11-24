package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidatePolicyName tests policy name validation (TDD)
func TestValidatePolicyName(t *testing.T) {
	tests := []struct {
		name       string
		policyName string
		wantErr    bool
		errMsg     string
	}{
		{"Valid simple name", "daily-backup", false, ""},
		{"Valid with underscores", "daily_backup_prod", false, ""},
		{"Valid with numbers", "backup-policy-123", false, ""},
		{"Valid mixed case", "DailyBackup", false, ""},
		{"Empty name", "", true, "name is required"},
		{"Too short", "ab", true, "name must be at least 3 characters"},
		{"Too long", string(make([]byte, 101)), true, "name must not exceed 100 characters"},
		{"Invalid characters - spaces", "daily backup", true, "name can only contain alphanumeric characters, hyphens, and underscores"},
		{"Invalid characters - special", "daily@backup", true, "name can only contain alphanumeric characters, hyphens, and underscores"},
		{"Invalid characters - dot", "daily.backup", true, "name can only contain alphanumeric characters, hyphens, and underscores"},
		{"Start with hyphen", "-backup", true, "name must start with an alphanumeric character"},
		{"End with hyphen", "backup-", true, "name must end with an alphanumeric character"},
		{"Unicode characters", "backup-日本語", true, "name can only contain alphanumeric characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePolicyName(tt.policyName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateCronSchedule tests cron expression validation (TDD)
func TestValidateCronSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantErr  bool
		errMsg   string
	}{
		{"Valid daily at 2am", "0 2 * * *", false, ""},
		{"Valid every 30 mins", "*/30 * * * *", false, ""},
		{"Valid weekly Sunday", "0 0 * * 0", false, ""},
		{"Valid monthly first day", "0 0 1 * *", false, ""},
		{"Valid specific time", "15 14 1 * *", false, ""},
		{"Valid range", "0 9-17 * * 1-5", false, ""},
		{"Valid list", "0 0,12 * * *", false, ""},
		{"Empty schedule", "", true, "schedule is required"},
		{"Invalid format - 4 fields", "0 2 * *", true, "invalid cron expression"},
		{"Invalid format - 6 fields", "0 0 2 * * *", true, "invalid cron expression"},
		{"Invalid minute", "60 2 * * *", true, "invalid minute"},
		{"Invalid hour", "0 24 * * *", true, "invalid hour"},
		{"Invalid day", "0 0 32 * *", true, "invalid day"},
		{"Invalid month", "0 0 1 13 *", true, "invalid month"},
		{"Invalid weekday", "0 0 * * 8", true, "invalid weekday"},
		{"Invalid characters", "a b c d e", true, "invalid cron expression"},
		{"Not a cron", "every day", true, "invalid cron expression"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCronSchedule(tt.schedule)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateIncludePaths tests include paths validation (TDD)
func TestValidateIncludePaths(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		wantErr bool
		errMsg  string
	}{
		{"Valid single path", []string{"/home"}, false, ""},
		{"Valid multiple paths", []string{"/home", "/var/www", "/etc"}, false, ""},
		{"Valid absolute paths", []string{"/data/backups", "/opt/app"}, false, ""},
		{"Empty array", []string{}, true, "at least one include path is required"},
		{"Nil array", nil, true, "at least one include path is required"},
		{"Too many paths", make([]string, 101), true, "cannot exceed 100 include paths"},
		{"Empty path string", []string{""}, true, "include path cannot be empty"},
		{"Relative path", []string{"home/user"}, true, "include path must be absolute"},
		{"Path too long", []string{"/" + string(make([]byte, 4097))}, true, "include path cannot exceed 4096 characters"},
		{"Windows absolute path", []string{"C:\\Users\\data"}, false, ""},
		{"Mixed valid paths", []string{"/home", "/var", "/opt"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create paths array if needed
			paths := tt.paths
			if tt.name == "Too many paths" {
				for i := 0; i < 101; i++ {
					paths[i] = "/path" + string(rune('0'+i%10))
				}
			}

			err := validateIncludePaths(paths)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateExcludePaths tests exclude paths validation (TDD)
func TestValidateExcludePaths(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		wantErr bool
		errMsg  string
	}{
		{"Valid patterns", []string{"*.tmp", "*.log", ".cache"}, false, ""},
		{"Valid directory patterns", []string{"*/node_modules", "*/.git"}, false, ""},
		{"Empty array - allowed", []string{}, false, ""},
		{"Nil array - allowed", nil, false, ""},
		{"Too many patterns", make([]string, 1001), true, "cannot exceed 1000 exclude paths"},
		{"Empty pattern string", []string{""}, true, "exclude path cannot be empty"},
		{"Pattern too long", []string{string(make([]byte, 4097))}, true, "exclude path cannot exceed 4096 characters"},
		{"Mixed patterns", []string{"*.tmp", "*/cache", ".DS_Store"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := tt.paths
			if tt.name == "Too many patterns" {
				for i := 0; i < 1001; i++ {
					paths[i] = "*.tmp" + string(rune('0'+i%10))
				}
			}

			err := validateExcludePaths(paths)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRepositoryType tests repository type validation (TDD)
func TestValidateRepositoryType(t *testing.T) {
	tests := []struct {
		name     string
		repoType string
		wantErr  bool
		errMsg   string
	}{
		{"Valid S3", "s3", false, ""},
		{"Valid REST server", "rest-server", false, ""},
		{"Valid filesystem", "fs", false, ""},
		{"Valid SFTP", "sftp", false, ""},
		{"Empty type", "", true, "repository type is required"},
		{"Invalid type", "ftp", true, "unsupported repository type"},
		{"Invalid type - unknown", "dropbox", true, "unsupported repository type"},
		{"Case sensitive", "S3", true, "unsupported repository type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepositoryType(tt.repoType)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateS3Repository tests S3 repository config validation (TDD)
func TestValidateS3Repository(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{"Valid minimal", map[string]interface{}{"type": "s3", "bucket": "my-bucket"}, false, ""},
		{"Valid with prefix", map[string]interface{}{"type": "s3", "bucket": "my-bucket", "prefix": "backups"}, false, ""},
		{"Valid with region", map[string]interface{}{"type": "s3", "bucket": "my-bucket", "region": "us-west-2"}, false, ""},
		{"Missing bucket", map[string]interface{}{"type": "s3"}, true, "S3 bucket is required"},
		{"Empty bucket", map[string]interface{}{"type": "s3", "bucket": ""}, true, "S3 bucket is required"},
		{"Invalid bucket name - uppercase", map[string]interface{}{"type": "s3", "bucket": "MyBucket"}, true, "invalid S3 bucket name"},
		{"Invalid bucket name - underscore", map[string]interface{}{"type": "s3", "bucket": "my_bucket"}, true, "invalid S3 bucket name"},
		{"Invalid bucket name - too short", map[string]interface{}{"type": "s3", "bucket": "ab"}, true, "bucket name must be 3-63 characters"},
		{"Invalid bucket name - too long", map[string]interface{}{"type": "s3", "bucket": string(make([]byte, 64))}, true, "bucket name must be 3-63 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Repository(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRestServerRepository tests REST server repository config validation (TDD)
func TestValidateRestServerRepository(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{"Valid HTTPS", map[string]interface{}{"type": "rest-server", "url": "https://backup.example.com:8000/repo"}, false, ""},
		{"Valid HTTP", map[string]interface{}{"type": "rest-server", "url": "http://localhost:8000/repo"}, false, ""},
		{"Missing URL", map[string]interface{}{"type": "rest-server"}, true, "REST server URL is required"},
		{"Empty URL", map[string]interface{}{"type": "rest-server", "url": ""}, true, "REST server URL is required"},
		{"Invalid URL - no scheme", map[string]interface{}{"type": "rest-server", "url": "backup.example.com"}, true, "REST server URL must use http or https"},
		{"Invalid URL - ftp scheme", map[string]interface{}{"type": "rest-server", "url": "ftp://backup.example.com"}, true, "REST server URL must use http or https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRestServerRepository(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFilesystemRepository tests filesystem repository config validation (TDD)
func TestValidateFilesystemRepository(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{"Valid absolute path", map[string]interface{}{"type": "fs", "path": "/mnt/backup/repo"}, false, ""},
		{"Valid Windows path", map[string]interface{}{"type": "fs", "path": "C:\\Backups\\repo"}, false, ""},
		{"Missing path", map[string]interface{}{"type": "fs"}, true, "filesystem path is required"},
		{"Empty path", map[string]interface{}{"type": "fs", "path": ""}, true, "filesystem path is required"},
		{"Relative path", map[string]interface{}{"type": "fs", "path": "backups/repo"}, true, "filesystem path must be absolute"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilesystemRepository(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateSFTPRepository tests SFTP repository config validation (TDD)
func TestValidateSFTPRepository(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{"Valid minimal", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup", "path": "/backups/repo"}, false, ""},
		{"Valid with port", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup", "path": "/backups/repo", "port": 2222}, false, ""},
		{"Missing host", map[string]interface{}{"type": "sftp", "user": "backup", "path": "/backups"}, true, "SFTP host is required"},
		{"Empty host", map[string]interface{}{"type": "sftp", "host": "", "user": "backup", "path": "/backups"}, true, "SFTP host is required"},
		{"Missing user", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "path": "/backups"}, true, "SFTP user is required"},
		{"Empty user", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "", "path": "/backups"}, true, "SFTP user is required"},
		{"Missing path", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup"}, true, "SFTP path is required"},
		{"Empty path", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup", "path": ""}, true, "SFTP path is required"},
		{"Invalid port - negative", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup", "path": "/backups", "port": -1}, true, "SFTP port must be between 1 and 65535"},
		{"Invalid port - too large", map[string]interface{}{"type": "sftp", "host": "backup.example.com", "user": "backup", "path": "/backups", "port": 65536}, true, "SFTP port must be between 1 and 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSFTPRepository(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRetentionRules tests retention rules validation (TDD)
func TestValidateRetentionRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{"Valid keep last", map[string]interface{}{"keepLast": 7}, false, ""},
		{"Valid keep daily", map[string]interface{}{"keepDaily": 14}, false, ""},
		{"Valid multiple rules", map[string]interface{}{"keepDaily": 7, "keepWeekly": 4, "keepMonthly": 12}, false, ""},
		{"Valid keep within", map[string]interface{}{"keepWithin": "30d"}, false, ""},
		{"Empty rules", map[string]interface{}{}, true, "at least one retention rule is required"},
		{"Nil rules", nil, true, "at least one retention rule is required"},
		{"Invalid keep last - zero", map[string]interface{}{"keepLast": 0}, true, "keepLast must be positive"},
		{"Invalid keep last - negative", map[string]interface{}{"keepLast": -5}, true, "keepLast must be positive"},
		{"Invalid keep daily - zero", map[string]interface{}{"keepDaily": 0}, true, "keepDaily must be positive"},
		{"Invalid keep weekly - negative", map[string]interface{}{"keepWeekly": -1}, true, "keepWeekly must be positive"},
		{"Invalid keep within format", map[string]interface{}{"keepWithin": "invalid"}, true, "invalid keepWithin format"},
		{"Invalid keep within - no unit", map[string]interface{}{"keepWithin": "30"}, true, "invalid keepWithin format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRetentionRules(tt.rules)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateBandwidthLimit tests bandwidth limit validation (TDD)
func TestValidateBandwidthLimit(t *testing.T) {
	tests := []struct {
		name    string
		limit   *int
		wantErr bool
		errMsg  string
	}{
		{"Valid limit", intPtr(10240), false, ""},
		{"Valid small limit", intPtr(1), false, ""},
		{"Valid large limit", intPtr(1000000), false, ""},
		{"Nil - allowed", nil, false, ""},
		{"Zero - invalid", intPtr(0), true, "bandwidth limit must be positive"},
		{"Negative - invalid", intPtr(-100), true, "bandwidth limit must be positive"},
		{"Too large", intPtr(1000001), true, "bandwidth limit cannot exceed 1000000 KB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBandwidthLimit(tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateParallelFiles tests parallel files validation (TDD)
func TestValidateParallelFiles(t *testing.T) {
	tests := []struct {
		name    string
		files   *int
		wantErr bool
		errMsg  string
	}{
		{"Valid default", intPtr(4), false, ""},
		{"Valid minimum", intPtr(1), false, ""},
		{"Valid maximum", intPtr(32), false, ""},
		{"Nil - allowed", nil, false, ""},
		{"Zero - invalid", intPtr(0), true, "parallel files must be between 1 and 32"},
		{"Negative - invalid", intPtr(-1), true, "parallel files must be between 1 and 32"},
		{"Too large", intPtr(33), true, "parallel files must be between 1 and 32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParallelFiles(tt.files)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePolicyRequest tests full policy request validation (TDD)
func TestValidatePolicyRequest(t *testing.T) {
	t.Run("Valid complete request", func(t *testing.T) {
		req := PolicyRequest{
			Name:         "daily-backup",
			Schedule:     "0 2 * * *",
			IncludePaths: []string{"/home", "/var/www"},
			ExcludePaths: []string{"*.tmp", "*.log"},
			Repository: map[string]interface{}{
				"type":   "s3",
				"bucket": "my-backups",
				"region": "us-west-2",
			},
			RetentionRules: map[string]interface{}{
				"keepDaily":   7,
				"keepWeekly":  4,
				"keepMonthly": 12,
			},
			BandwidthLimitKBps: intPtr(10240),
			ParallelFiles:      intPtr(4),
		}

		err := validatePolicyRequest(&req)
		assert.NoError(t, err)
	})

	t.Run("Invalid - multiple validation errors", func(t *testing.T) {
		req := PolicyRequest{
			Name:         "a",        // Too short
			Schedule:     "invalid",  // Invalid cron
			IncludePaths: []string{}, // Empty
			Repository: map[string]interface{}{
				"type": "unknown", // Invalid type
			},
			RetentionRules:     map[string]interface{}{}, // Empty
			BandwidthLimitKBps: intPtr(-100),             // Negative
			ParallelFiles:      intPtr(100),              // Too large
		}

		err := validatePolicyRequest(&req)
		assert.Error(t, err)
		// Should return first error encountered
	})
}
