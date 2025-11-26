package agent

import (
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryInfo contains retry tracking information for a task
type RetryInfo struct {
	RetryCount  int
	MaxRetries  int
	LastError   string
	NextRetryAt *time.Time
}

// CalculateExponentialBackoff calculates backoff duration with exponential growth and optional jitter
// attempt: retry attempt number (1-based)
// baseDelay: initial delay for first retry
// maxDelay: maximum delay cap
// jitter: jitter factor (0.0 to 1.0, where 1.0 = 100% jitter)
func CalculateExponentialBackoff(attempt int, baseDelay, maxDelay time.Duration, jitter float64) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	// Calculate exponential backoff: base * 2^(attempt-1)
	exponent := attempt - 1
	multiplier := math.Pow(2, float64(exponent))
	backoff := time.Duration(float64(baseDelay) * multiplier)

	// Cap at maximum
	if backoff > maxDelay {
		backoff = maxDelay
	}

	// Apply jitter if requested
	if jitter > 0 {
		backoff = applyJitter(backoff, jitter)
	}

	return backoff
}

// applyJitter adds randomness to backoff to prevent thundering herd
// jitter factor determines range: 0.5 means ±50% of base value
func applyJitter(base time.Duration, jitter float64) time.Duration {
	if jitter == 0 {
		return base
	}

	// Calculate jitter range
	jitterAmount := float64(base) * jitter
	
	// Random value between -jitterAmount and +jitterAmount
	randomJitter := (rand.Float64()*2 - 1) * jitterAmount
	
	result := time.Duration(float64(base) + randomJitter)
	
	// Ensure we don't go negative
	if result < 0 {
		result = 0
	}
	
	return result
}

// CalculateNextRetryTime calculates when the next retry should occur
func CalculateNextRetryTime(retryCount int, baseDelay, maxDelay time.Duration, jitter float64, now time.Time) time.Time {
	backoff := CalculateExponentialBackoff(retryCount, baseDelay, maxDelay, jitter)
	return now.Add(backoff)
}

// ShouldRetryTask determines if a task should be retried based on retry info
// ShouldRetryTask determines if a task should be retried based on retry info
func ShouldRetryTask(info RetryInfo) (bool, string) {
	// Check if max retries reached
	if info.RetryCount >= info.MaxRetries {
		log.Printf("[BACKOFF] Task exhausted: retry_count=%d, max_retries=%d", info.RetryCount, info.MaxRetries)
		return false, "max retries reached"
	}

	// Check for permanent errors that shouldn't be retried
	if isPermanentError(info.LastError) {
		log.Printf("[BACKOFF] Permanent failure detected: error=%q", info.LastError)
		return false, "permanent error"
	}

	// Check if still in backoff period
	if info.NextRetryAt != nil && time.Now().Before(*info.NextRetryAt) {
		return false, "still in backoff period"
	}

	return true, ""
}

// isPermanentError checks if an error is permanent and shouldn't be retried
func isPermanentError(errorMsg string) bool {
	permanentErrors := []string{
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
		"not found",
		"invalid repository",
		"authentication failed",
	}

	lowerError := strings.ToLower(errorMsg)
	for _, permanent := range permanentErrors {
		if strings.Contains(lowerError, permanent) {
			return true
		}
	}
	return false
}

// UpdateRetryInfo updates retry information after a task failure
func UpdateRetryInfo(current RetryInfo, errorMsg string, baseDelay, maxDelay time.Duration, jitter float64) RetryInfo {
	newRetryCount := current.RetryCount + 1
	nextRetry := CalculateNextRetryTime(newRetryCount, baseDelay, maxDelay, jitter, time.Now())
	
	// Log backoff event with structured information
	log.Printf("[BACKOFF] Task entering backoff: retry=%d/%d, next_retry=%v, delay=%v, error=%q",
		newRetryCount, current.MaxRetries, nextRetry.Format(time.RFC3339), time.Until(nextRetry), errorMsg)

	return RetryInfo{
		RetryCount:  newRetryCount,
		MaxRetries:  current.MaxRetries,
		LastError:   errorMsg,
		NextRetryAt: &nextRetry,
	}
}

// ResetRetryInfo creates a fresh RetryInfo for a new task or after success
func ResetRetryInfo(maxRetries int) RetryInfo {
	return RetryInfo{
		RetryCount:  0,
		MaxRetries:  maxRetries,
		LastError:   "",
		NextRetryAt: nil,
	}
}
