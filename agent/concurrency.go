package agent

import (
	"fmt"
	"log"
)

// ConcurrencyConfig defines per-agent concurrency and quota settings
type ConcurrencyConfig struct {
	MaxConcurrentTasks   int  `yaml:"maxConcurrentTasks" json:"maxConcurrentTasks"`
	MaxConcurrentBackups int  `yaml:"maxConcurrentBackups" json:"maxConcurrentBackups"`
	MaxConcurrentChecks  int  `yaml:"maxConcurrentChecks" json:"maxConcurrentChecks"`
	MaxConcurrentPrunes  int  `yaml:"maxConcurrentPrunes" json:"maxConcurrentPrunes"`
	CPUQuotaPercent      int  `yaml:"cpuQuotaPercent" json:"cpuQuotaPercent"`
	BandwidthLimitMbps   *int `yaml:"bandwidthLimitMbps,omitempty" json:"bandwidthLimitMbps,omitempty"`
}

// ApplyConcurrencyDefaults sets default values for concurrency configuration
func ApplyConcurrencyDefaults(cfg *ConcurrencyConfig) {
	// Set default total tasks first
	if cfg.MaxConcurrentTasks == 0 {
		cfg.MaxConcurrentTasks = 3 // Default: allow 3 total concurrent tasks
	}

	// Set per-type defaults only if they're zero
	// These should sum to <= MaxConcurrentTasks
	if cfg.MaxConcurrentBackups == 0 {
		cfg.MaxConcurrentBackups = 1
	}
	if cfg.MaxConcurrentChecks == 0 {
		cfg.MaxConcurrentChecks = 1
	}
	if cfg.MaxConcurrentPrunes == 0 {
		cfg.MaxConcurrentPrunes = 1
	}

	if cfg.CPUQuotaPercent == 0 {
		cfg.CPUQuotaPercent = 50
	}
}

// ValidateConcurrencyConfig validates concurrency configuration values
func ValidateConcurrencyConfig(cfg *ConcurrencyConfig) error {
	// Validate MaxConcurrentTasks
	if cfg.MaxConcurrentTasks <= 0 {
		return fmt.Errorf("maxConcurrentTasks must be positive")
	}
	if cfg.MaxConcurrentTasks > 100 {
		return fmt.Errorf("maxConcurrentTasks cannot exceed 100")
	}

	// Validate per-type limits
	if cfg.MaxConcurrentBackups < 0 {
		return fmt.Errorf("maxConcurrentBackups must be non-negative")
	}
	if cfg.MaxConcurrentChecks < 0 {
		return fmt.Errorf("maxConcurrentChecks must be non-negative")
	}
	if cfg.MaxConcurrentPrunes < 0 {
		return fmt.Errorf("maxConcurrentPrunes must be non-negative")
	}

	// Validate that sum of per-type limits doesn't exceed total
	sum := cfg.MaxConcurrentBackups + cfg.MaxConcurrentChecks + cfg.MaxConcurrentPrunes
	if sum > cfg.MaxConcurrentTasks {
		return fmt.Errorf("sum of per-type limits (%d) exceeds total limit (%d)", sum, cfg.MaxConcurrentTasks)
	}

	// Validate CPU quota
	if cfg.CPUQuotaPercent < 1 || cfg.CPUQuotaPercent > 100 {
		return fmt.Errorf("cpuQuotaPercent must be between 1 and 100")
	}

	// Validate bandwidth limit if set
	if cfg.BandwidthLimitMbps != nil {
		if *cfg.BandwidthLimitMbps <= 0 {
			return fmt.Errorf("bandwidthLimitMbps must be positive if set")
		}
		if *cfg.BandwidthLimitMbps > 100000 {
			return fmt.Errorf("bandwidthLimitMbps cannot exceed 100000")
		}
	}

	return nil
}

// MergeConcurrencyConfig merges two concurrency configs, with updates taking precedence
func MergeConcurrencyConfig(base, updates *ConcurrencyConfig) *ConcurrencyConfig {
	merged := &ConcurrencyConfig{
		MaxConcurrentTasks:   base.MaxConcurrentTasks,
		MaxConcurrentBackups: base.MaxConcurrentBackups,
		MaxConcurrentChecks:  base.MaxConcurrentChecks,
		MaxConcurrentPrunes:  base.MaxConcurrentPrunes,
		CPUQuotaPercent:      base.CPUQuotaPercent,
		BandwidthLimitMbps:   base.BandwidthLimitMbps,
	}

	if updates.MaxConcurrentTasks > 0 {
		merged.MaxConcurrentTasks = updates.MaxConcurrentTasks
	}
	if updates.MaxConcurrentBackups > 0 {
		merged.MaxConcurrentBackups = updates.MaxConcurrentBackups
	}
	if updates.MaxConcurrentChecks > 0 {
		merged.MaxConcurrentChecks = updates.MaxConcurrentChecks
	}
	if updates.MaxConcurrentPrunes > 0 {
		merged.MaxConcurrentPrunes = updates.MaxConcurrentPrunes
	}
	if updates.CPUQuotaPercent > 0 {
		merged.CPUQuotaPercent = updates.CPUQuotaPercent
	}
	if updates.BandwidthLimitMbps != nil {
		merged.BandwidthLimitMbps = updates.BandwidthLimitMbps
	}

	return merged
}

// ConcurrencyLimiter limits the number of concurrent task executions
type ConcurrencyLimiter struct {
	semaphore chan struct{}
}

// NewConcurrencyLimiter creates a new concurrency limiter
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a slot for task execution (blocks if limit reached)
func (cl *ConcurrencyLimiter) Acquire() {
	// Check if we're at capacity before blocking
	if len(cl.semaphore) == cap(cl.semaphore) {
		log.Printf("[CONCURRENCY] Concurrency limit reached (%d/%d), waiting for slot...", len(cl.semaphore), cap(cl.semaphore))
	}
	cl.semaphore <- struct{}{}
}

// Release releases a slot after task execution
func (cl *ConcurrencyLimiter) Release() {
	<-cl.semaphore
}

// Available returns the number of available slots
func (cl *ConcurrencyLimiter) Available() int {
	return cap(cl.semaphore) - len(cl.semaphore)
}
