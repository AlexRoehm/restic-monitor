package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoff(t *testing.T) {
	t.Run("Calculate backoff for first retry", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(1, 5*time.Second, 60*time.Second, 0)

		// First retry: base * 2^0 = 5 seconds
		assert.Equal(t, 5*time.Second, backoff)
	})

	t.Run("Calculate backoff for second retry", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(2, 5*time.Second, 60*time.Second, 0)

		// Second retry: base * 2^1 = 10 seconds
		assert.Equal(t, 10*time.Second, backoff)
	})

	t.Run("Calculate backoff for third retry", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(3, 5*time.Second, 60*time.Second, 0)

		// Third retry: base * 2^2 = 20 seconds
		assert.Equal(t, 20*time.Second, backoff)
	})

	t.Run("Respect maximum backoff", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(10, 5*time.Second, 60*time.Second, 0)

		// Would be 5 * 2^9 = 2560 seconds, but capped at 60
		assert.Equal(t, 60*time.Second, backoff)
	})

	t.Run("Add jitter to backoff", func(t *testing.T) {
		jitter := 2.0 // 200% jitter

		// Run multiple times to test randomness
		for i := 0; i < 10; i++ {
			backoff := CalculateExponentialBackoff(2, 5*time.Second, 60*time.Second, jitter)

			// With 200% jitter, backoff should be between 0 and 30 seconds (10s * 3)
			assert.GreaterOrEqual(t, backoff, time.Duration(0))
			assert.LessOrEqual(t, backoff, 30*time.Second)
		}
	})

	t.Run("Zero attempt returns base", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(0, 5*time.Second, 60*time.Second, 0)

		assert.Equal(t, 5*time.Second, backoff)
	})

	t.Run("Negative attempt returns base", func(t *testing.T) {
		backoff := CalculateExponentialBackoff(-1, 5*time.Second, 60*time.Second, 0)

		assert.Equal(t, 5*time.Second, backoff)
	})
}

func TestShouldRetryTask(t *testing.T) {
	t.Run("Retry transient error", func(t *testing.T) {
		retryInfo := RetryInfo{
			RetryCount:  2,
			MaxRetries:  5,
			LastError:   "connection timeout",
			NextRetryAt: nil,
		}

		should, reason := ShouldRetryTask(retryInfo)
		assert.True(t, should)
		assert.Empty(t, reason)
	})

	t.Run("Do not retry when max retries reached", func(t *testing.T) {
		retryInfo := RetryInfo{
			RetryCount:  5,
			MaxRetries:  5,
			LastError:   "connection timeout",
			NextRetryAt: nil,
		}

		should, reason := ShouldRetryTask(retryInfo)
		assert.False(t, should)
		assert.Contains(t, reason, "max retries")
	})

	t.Run("Do not retry when in backoff period", func(t *testing.T) {
		futureTime := time.Now().Add(10 * time.Minute)
		retryInfo := RetryInfo{
			RetryCount:  2,
			MaxRetries:  5,
			LastError:   "connection timeout",
			NextRetryAt: &futureTime,
		}

		should, reason := ShouldRetryTask(retryInfo)
		assert.False(t, should)
		assert.Contains(t, reason, "backoff")
	})

	t.Run("Retry after backoff period expires", func(t *testing.T) {
		pastTime := time.Now().Add(-10 * time.Minute)
		retryInfo := RetryInfo{
			RetryCount:  2,
			MaxRetries:  5,
			LastError:   "connection timeout",
			NextRetryAt: &pastTime,
		}

		should, reason := ShouldRetryTask(retryInfo)
		assert.True(t, should)
		assert.Empty(t, reason)
	})

	t.Run("Do not retry permanent error", func(t *testing.T) {
		retryInfo := RetryInfo{
			RetryCount:  1,
			MaxRetries:  5,
			LastError:   "permission denied",
			NextRetryAt: nil,
		}

		should, reason := ShouldRetryTask(retryInfo)
		assert.False(t, should)
		assert.Contains(t, reason, "permanent")
	})
}

func TestCalculateNextRetryTime(t *testing.T) {
	t.Run("Calculate next retry with backoff", func(t *testing.T) {
		now := time.Now()
		retryCount := 2

		nextRetry := CalculateNextRetryTime(retryCount, 5*time.Second, 60*time.Second, 0.5, now)

		// Should be in the future
		assert.True(t, nextRetry.After(now))

		// Should be approximately 10 seconds from now (base * 2^(attempt-1))
		// With 50% jitter: between 5 and 15 seconds
		duration := nextRetry.Sub(now)
		assert.GreaterOrEqual(t, duration, 5*time.Second)
		assert.LessOrEqual(t, duration, 15*time.Second)
	})

	t.Run("First retry has minimum backoff", func(t *testing.T) {
		now := time.Now()

		nextRetry := CalculateNextRetryTime(1, 5*time.Second, 60*time.Second, 0, now)

		// First retry should be exactly 5 seconds
		expected := now.Add(5 * time.Second)
		assert.Equal(t, expected.Unix(), nextRetry.Unix())
	})

	t.Run("Respect maximum backoff", func(t *testing.T) {
		now := time.Now()

		nextRetry := CalculateNextRetryTime(10, 5*time.Second, 30*time.Second, 0, now)

		// Should be capped at max backoff
		expected := now.Add(30 * time.Second)
		assert.Equal(t, expected.Unix(), nextRetry.Unix())
	})
}

func TestRetryInfoUpdate(t *testing.T) {
	t.Run("Update retry info after failure", func(t *testing.T) {
		retryInfo := RetryInfo{
			RetryCount:  1,
			MaxRetries:  5,
			LastError:   "",
			NextRetryAt: nil,
		}

		newError := "network timeout"
		updated := UpdateRetryInfo(retryInfo, newError, 5*time.Second, 60*time.Second, 0.5)

		assert.Equal(t, 2, updated.RetryCount)
		assert.Equal(t, newError, updated.LastError)
		assert.NotNil(t, updated.NextRetryAt)
		assert.True(t, updated.NextRetryAt.After(time.Now()))
	})

	t.Run("Reset retry info after success", func(t *testing.T) {
		reset := ResetRetryInfo(5)

		assert.Equal(t, 0, reset.RetryCount)
		assert.Equal(t, 5, reset.MaxRetries)
		assert.Empty(t, reset.LastError)
		assert.Nil(t, reset.NextRetryAt)
	})
}

func TestBackoffWithJitter(t *testing.T) {
	t.Run("Jitter produces different values", func(t *testing.T) {
		base := 10 * time.Second
		jitter := 1.0 // 100% jitter

		values := make(map[time.Duration]bool)
		for i := 0; i < 20; i++ {
			result := applyJitter(base, jitter)
			values[result] = true
		}

		// With randomness, we should see multiple different values
		assert.Greater(t, len(values), 1, "Jitter should produce varied results")
	})

	t.Run("Zero jitter produces consistent value", func(t *testing.T) {
		base := 10 * time.Second
		jitter := 0.0

		for i := 0; i < 10; i++ {
			result := applyJitter(base, jitter)
			assert.Equal(t, base, result)
		}
	})

	t.Run("Jitter stays within bounds", func(t *testing.T) {
		base := 10 * time.Second
		jitter := 0.5 // 50% jitter

		for i := 0; i < 100; i++ {
			result := applyJitter(base, jitter)

			// With 50% jitter: between 5s and 15s
			assert.GreaterOrEqual(t, result, 5*time.Second)
			assert.LessOrEqual(t, result, 15*time.Second)
		}
	})
}
