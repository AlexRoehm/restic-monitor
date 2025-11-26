package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrencyConfigDefaults(t *testing.T) {
	cfg := &ConcurrencyConfig{}
	ApplyConcurrencyDefaults(cfg)

	assert.Equal(t, 3, cfg.MaxConcurrentTasks, "Default max concurrent tasks should be 3")
	assert.Equal(t, 1, cfg.MaxConcurrentBackups, "Default max concurrent backups should be 1")
	assert.Equal(t, 1, cfg.MaxConcurrentChecks, "Default max concurrent checks should be 1")
	assert.Equal(t, 1, cfg.MaxConcurrentPrunes, "Default max concurrent prunes should be 1")
	assert.Equal(t, 50, cfg.CPUQuotaPercent, "Default CPU quota should be 50%")
	assert.Nil(t, cfg.BandwidthLimitMbps, "Default bandwidth limit should be nil (unlimited)")
}

func TestValidateConcurrencyConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ConcurrencyConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid config with all fields",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      75,
				BandwidthLimitMbps:   intPtr(100),
			},
			expectErr: false,
		},
		{
			name: "Valid config with defaults",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      50,
			},
			expectErr: false,
		},
		{
			name: "Invalid - negative max concurrent tasks",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   -1,
				MaxConcurrentBackups: 0,
				MaxConcurrentChecks:  0,
				MaxConcurrentPrunes:  0,
				CPUQuotaPercent:      50,
			},
			expectErr: true,
			errMsg:    "maxConcurrentTasks must be positive",
		},
		{
			name: "Invalid - zero max concurrent tasks",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   0,
				MaxConcurrentBackups: 0,
				MaxConcurrentChecks:  0,
				MaxConcurrentPrunes:  0,
				CPUQuotaPercent:      50,
			},
			expectErr: true,
			errMsg:    "maxConcurrentTasks must be positive",
		},
		{
			name: "Invalid - max concurrent tasks too high",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   101,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      50,
			},
			expectErr: true,
			errMsg:    "maxConcurrentTasks cannot exceed 100",
		},
		{
			name: "Invalid - negative max concurrent backups",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   1,
				MaxConcurrentBackups: -1,
				MaxConcurrentChecks:  0,
				MaxConcurrentPrunes:  0,
				CPUQuotaPercent:      50,
			},
			expectErr: true,
			errMsg:    "maxConcurrentBackups must be non-negative",
		},
		{
			name: "Invalid - CPU quota too low",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      0,
			},
			expectErr: true,
			errMsg:    "cpuQuotaPercent must be between 1 and 100",
		},
		{
			name: "Invalid - CPU quota too high",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      101,
			},
			expectErr: true,
			errMsg:    "cpuQuotaPercent must be between 1 and 100",
		},
		{
			name: "Invalid - negative bandwidth limit",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      50,
				BandwidthLimitMbps:   intPtr(-1),
			},
			expectErr: true,
			errMsg:    "bandwidthLimitMbps must be positive if set",
		},
		{
			name: "Invalid - bandwidth limit too high",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      50,
				BandwidthLimitMbps:   intPtr(100001),
			},
			expectErr: true,
			errMsg:    "bandwidthLimitMbps cannot exceed 100000",
		},
		{
			name: "Invalid - per-type sum exceeds total",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   2,
				MaxConcurrentBackups: 2,
				MaxConcurrentChecks:  2,
				MaxConcurrentPrunes:  2,
				CPUQuotaPercent:      50,
			},
			expectErr: true,
			errMsg:    "exceeds total limit",
		},
		{
			name: "Valid - per-type sum equals total",
			config: ConcurrencyConfig{
				MaxConcurrentTasks:   3,
				MaxConcurrentBackups: 1,
				MaxConcurrentChecks:  1,
				MaxConcurrentPrunes:  1,
				CPUQuotaPercent:      50,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConcurrencyConfig(&tt.config)
			if tt.expectErr {
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

func TestConcurrencyConfigMerge(t *testing.T) {
	base := &ConcurrencyConfig{
		MaxConcurrentTasks:   1,
		MaxConcurrentBackups: 1,
		MaxConcurrentChecks:  1,
		MaxConcurrentPrunes:  1,
		CPUQuotaPercent:      50,
	}

	updates := &ConcurrencyConfig{
		MaxConcurrentTasks: 2,
		CPUQuotaPercent:    75,
		BandwidthLimitMbps: intPtr(100),
	}

	merged := MergeConcurrencyConfig(base, updates)

	assert.Equal(t, 2, merged.MaxConcurrentTasks, "Should update maxConcurrentTasks")
	assert.Equal(t, 1, merged.MaxConcurrentBackups, "Should keep original maxConcurrentBackups")
	assert.Equal(t, 75, merged.CPUQuotaPercent, "Should update CPUQuotaPercent")
	assert.NotNil(t, merged.BandwidthLimitMbps, "Should add BandwidthLimitMbps")
	assert.Equal(t, 100, *merged.BandwidthLimitMbps)
}

// Helper function
func intPtr(i int) *int {
	return &i
}
