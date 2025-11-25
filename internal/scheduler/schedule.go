package scheduler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"
	ScheduleTypeInterval ScheduleType = "interval"
)

// ParsedSchedule represents a parsed schedule before normalization
type ParsedSchedule struct {
	Type     ScheduleType
	Cron     string
	Interval time.Duration
}

// NormalizedSchedule represents a schedule in standard form
type NormalizedSchedule struct {
	Type     ScheduleType  `json:"type"`
	Cron     string        `json:"cron,omitempty"`
	Interval time.Duration `json:"interval,omitempty"`
}

var intervalRegex = regexp.MustCompile(`^every\s+(\d+)([hm])$`)

// ParseSchedule parses a schedule string into a ParsedSchedule
func ParseSchedule(schedule string) (*ParsedSchedule, error) {
	if schedule == "" {
		return nil, fmt.Errorf("schedule cannot be empty")
	}

	// Try parsing as interval first
	if strings.HasPrefix(schedule, "every ") {
		return parseInterval(schedule)
	}

	// Try parsing as cron
	return parseCron(schedule)
}

// parseInterval parses "every Xh" or "every Xm" format
func parseInterval(schedule string) (*ParsedSchedule, error) {
	matches := intervalRegex.FindStringSubmatch(schedule)
	if matches == nil {
		return nil, fmt.Errorf("invalid interval format: %s", schedule)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid interval value: %s", matches[1])
	}

	var duration time.Duration
	switch matches[2] {
	case "h":
		duration = time.Duration(value) * time.Hour
	case "m":
		duration = time.Duration(value) * time.Minute
	default:
		return nil, fmt.Errorf("invalid interval unit: %s", matches[2])
	}

	return &ParsedSchedule{
		Type:     ScheduleTypeInterval,
		Interval: duration,
	}, nil
}

// parseCron validates and parses a cron expression
func parseCron(schedule string) (*ParsedSchedule, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	return &ParsedSchedule{
		Type: ScheduleTypeCron,
		Cron: schedule,
	}, nil
}

// NormalizeSchedule converts a ParsedSchedule to NormalizedSchedule
func NormalizeSchedule(parsed *ParsedSchedule) NormalizedSchedule {
	return NormalizedSchedule{
		Type:     parsed.Type,
		Cron:     parsed.Cron,
		Interval: parsed.Interval,
	}
}

// ValidateSchedule validates a schedule string
func ValidateSchedule(schedule string) error {
	_, err := ParseSchedule(schedule)
	return err
}

// ComputeNextRun computes the next run time for a schedule
func ComputeNextRun(schedule NormalizedSchedule, now time.Time) (time.Time, error) {
	switch schedule.Type {
	case ScheduleTypeCron:
		return computeNextRunCron(schedule.Cron, now)
	case ScheduleTypeInterval:
		// For interval without last run, use now + interval
		return now.Add(schedule.Interval), nil
	default:
		return time.Time{}, fmt.Errorf("unknown schedule type: %s", schedule.Type)
	}
}

// ComputeNextRunWithLast computes next run time considering last run
func ComputeNextRunWithLast(schedule NormalizedSchedule, now time.Time, lastRun *time.Time) (time.Time, error) {
	switch schedule.Type {
	case ScheduleTypeCron:
		return computeNextRunCron(schedule.Cron, now)
	case ScheduleTypeInterval:
		if lastRun == nil {
			return now.Add(schedule.Interval), nil
		}
		next := lastRun.Add(schedule.Interval)
		// If next is in the past, use now + interval
		if next.Before(now) {
			return now.Add(schedule.Interval), nil
		}
		return next, nil
	default:
		return time.Time{}, fmt.Errorf("unknown schedule type: %s", schedule.Type)
	}
}

// computeNextRunCron computes next run for cron expression
func computeNextRunCron(cronExpr string, now time.Time) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	return schedule.Next(now), nil
}
