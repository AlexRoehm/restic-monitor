package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseValidCronSchedule tests parsing valid cron expressions (TDD - Epic 14.1)
func TestParseValidCronSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantType ScheduleType
	}{
		{
			name:     "daily at 2am",
			schedule: "0 2 * * *",
			wantType: ScheduleTypeCron,
		},
		{
			name:     "hourly",
			schedule: "0 * * * *",
			wantType: ScheduleTypeCron,
		},
		{
			name:     "every 15 minutes",
			schedule: "*/15 * * * *",
			wantType: ScheduleTypeCron,
		},
		{
			name:     "weekly on Monday",
			schedule: "0 3 * * 1",
			wantType: ScheduleTypeCron,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSchedule(tt.schedule)
			require.NoError(t, err, "Should parse valid cron expression")
			assert.Equal(t, tt.wantType, parsed.Type)
			assert.Equal(t, tt.schedule, parsed.Cron)
		})
	}
}

// TestParseValidIntervalSchedule tests parsing interval-based schedules (TDD - Epic 14.1)
func TestParseValidIntervalSchedule(t *testing.T) {
	tests := []struct {
		name         string
		schedule     string
		wantInterval time.Duration
	}{
		{
			name:         "every 6 hours",
			schedule:     "every 6h",
			wantInterval: 6 * time.Hour,
		},
		{
			name:         "every 30 minutes",
			schedule:     "every 30m",
			wantInterval: 30 * time.Minute,
		},
		{
			name:         "every 24 hours",
			schedule:     "every 24h",
			wantInterval: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSchedule(tt.schedule)
			require.NoError(t, err, "Should parse valid interval")
			assert.Equal(t, ScheduleTypeInterval, parsed.Type)
			assert.Equal(t, tt.wantInterval, parsed.Interval)
		})
	}
}

// TestParseInvalidSchedule tests rejection of invalid schedules (TDD - Epic 14.1)
func TestParseInvalidSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
	}{
		{
			name:     "empty string",
			schedule: "",
		},
		{
			name:     "invalid cron",
			schedule: "not a cron",
		},
		{
			name:     "invalid interval format",
			schedule: "every foo",
		},
		{
			name:     "interval without unit",
			schedule: "every 5",
		},
		{
			name:     "too many fields",
			schedule: "* * * * * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSchedule(tt.schedule)
			assert.Error(t, err, "Should reject invalid schedule")
		})
	}
}

// TestNormalizeSchedule tests schedule normalization (TDD - Epic 14.2)
func TestNormalizeSchedule(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     NormalizedSchedule
	}{
		{
			name:     "cron expression",
			schedule: "0 2 * * *",
			want: NormalizedSchedule{
				Type: ScheduleTypeCron,
				Cron: "0 2 * * *",
			},
		},
		{
			name:     "interval expression",
			schedule: "every 6h",
			want: NormalizedSchedule{
				Type:     ScheduleTypeInterval,
				Interval: 6 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseSchedule(tt.schedule)
			require.NoError(t, err)

			normalized := NormalizeSchedule(parsed)
			assert.Equal(t, tt.want.Type, normalized.Type)

			if tt.want.Type == ScheduleTypeCron {
				assert.Equal(t, tt.want.Cron, normalized.Cron)
			} else {
				assert.Equal(t, tt.want.Interval, normalized.Interval)
			}
		})
	}
}

// TestComputeNextRunCron tests computing next run for cron schedules (TDD - Epic 14.2)
func TestComputeNextRunCron(t *testing.T) {
	// Fixed time: 2025-01-15 10:30:00
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		cron     string
		wantHour int
		wantMin  int
	}{
		{
			name:     "daily at 2am - next occurrence",
			cron:     "0 2 * * *",
			wantHour: 2,
			wantMin:  0,
		},
		{
			name:     "every hour - next occurrence",
			cron:     "0 * * * *",
			wantHour: 11,
			wantMin:  0,
		},
		{
			name:     "every 15 minutes",
			cron:     "*/15 * * * *",
			wantHour: 10,
			wantMin:  45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := NormalizedSchedule{
				Type: ScheduleTypeCron,
				Cron: tt.cron,
			}

			next, err := ComputeNextRun(schedule, now)
			require.NoError(t, err)
			assert.True(t, next.After(now), "Next run should be in the future")
			assert.Equal(t, tt.wantHour, next.Hour())
			assert.Equal(t, tt.wantMin, next.Minute())
		})
	}
}

// TestComputeNextRunInterval tests computing next run for interval schedules (TDD - Epic 14.2)
func TestComputeNextRunInterval(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		interval time.Duration
		lastRun  *time.Time
		wantNext time.Time
	}{
		{
			name:     "6 hours from now (no last run)",
			interval: 6 * time.Hour,
			lastRun:  nil,
			wantNext: now.Add(6 * time.Hour),
		},
		{
			name:     "6 hours from last run",
			interval: 6 * time.Hour,
			lastRun:  &[]time.Time{now.Add(-4 * time.Hour)}[0],
			wantNext: now.Add(2 * time.Hour),
		},
		{
			name:     "30 minutes from now",
			interval: 30 * time.Minute,
			lastRun:  nil,
			wantNext: now.Add(30 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := NormalizedSchedule{
				Type:     ScheduleTypeInterval,
				Interval: tt.interval,
			}

			next, err := ComputeNextRunWithLast(schedule, now, tt.lastRun)
			require.NoError(t, err)
			assert.Equal(t, tt.wantNext, next)
		})
	}
}

// TestValidateScheduleFormat tests schedule validation (TDD - Epic 14.1)
func TestValidateScheduleFormat(t *testing.T) {
	tests := []struct {
		name      string
		schedule  string
		wantValid bool
	}{
		{
			name:      "valid cron",
			schedule:  "0 2 * * *",
			wantValid: true,
		},
		{
			name:      "valid interval",
			schedule:  "every 6h",
			wantValid: true,
		},
		{
			name:      "invalid format",
			schedule:  "invalid",
			wantValid: false,
		},
		{
			name:      "empty",
			schedule:  "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchedule(tt.schedule)
			if tt.wantValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
