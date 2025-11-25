package agent

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
