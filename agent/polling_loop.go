package agent

import (
	"context"
	"fmt"
	"log"
	"time"
)

// PollingLoop manages the agent's main polling cycle
type PollingLoop struct {
	config      *Config
	state       *State
	heartbeat   *HeartbeatClient
	taskClient  *TaskClient
	queue       *TaskQueue
	metrics     *LoopMetrics
	version     string
	stopChan    chan struct{}
	stoppedChan chan struct{}
	logPrefix   string
}

// NewPollingLoop creates a new polling loop
func NewPollingLoop(cfg *Config, state *State, version string) *PollingLoop {
	return &PollingLoop{
		config:      cfg,
		state:       state,
		heartbeat:   NewHeartbeatClient(cfg, state, version),
		taskClient:  NewTaskClient(cfg, state),
		queue:       NewTaskQueue(),
		metrics:     NewLoopMetrics(),
		version:     version,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
		logPrefix:   "[PollingLoop]",
	}
}

// Start begins the polling loop
func (p *PollingLoop) Start(ctx context.Context) error {
	log.Printf("%s Starting polling loop (interval: %ds)", p.logPrefix, p.config.PollingIntervalSeconds)

	ticker := time.NewTicker(time.Duration(p.config.PollingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run initial iteration immediately
	p.runIteration()

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s Context cancelled, stopping", p.logPrefix)
			close(p.stoppedChan)
			return ctx.Err()
		case <-p.stopChan:
			log.Printf("%s Stop signal received, stopping", p.logPrefix)
			close(p.stoppedChan)
			return nil
		case <-ticker.C:
			p.runIteration()
		}
	}
}

// Stop stops the polling loop gracefully
func (p *PollingLoop) Stop() {
	close(p.stopChan)
	<-p.stoppedChan
}

// runIteration executes one iteration of the polling loop
func (p *PollingLoop) runIteration() {
	startTime := time.Now()
	p.metrics.IncrementLoopCount()

	loopNum := p.metrics.GetLoopCount()
	log.Printf("%s Loop iteration #%d started", p.logPrefix, loopNum)

	// Send heartbeat
	p.sendHeartbeat()

	// Fetch pending tasks
	p.fetchTasks()

	// Record loop duration
	duration := time.Since(startTime)
	p.metrics.RecordLoopDuration(duration)

	log.Printf("%s Loop iteration #%d completed in %v", p.logPrefix, loopNum, duration)
}

// sendHeartbeat sends a heartbeat to the orchestrator
func (p *PollingLoop) sendHeartbeat() {
	log.Printf("%s Sending heartbeat", p.logPrefix)

	err := p.heartbeat.SendHeartbeat()
	if err != nil {
		log.Printf("%s Heartbeat failed: %v", p.logPrefix, err)
		p.metrics.RecordHeartbeatError(err)
		return
	}

	log.Printf("%s Heartbeat successful", p.logPrefix)
	p.metrics.RecordHeartbeatSuccess()
}

// fetchTasks retrieves pending tasks from the orchestrator
func (p *PollingLoop) fetchTasks() {
	log.Printf("%s Fetching pending tasks", p.logPrefix)

	tasks, err := p.taskClient.FetchTasks()
	if err != nil {
		log.Printf("%s Task fetch failed: %v", p.logPrefix, err)
		p.metrics.RecordTaskFetchError(err)
		return
	}

	taskCount := len(tasks)
	log.Printf("%s Fetched %d task(s)", p.logPrefix, taskCount)
	p.metrics.RecordTasksFetched(taskCount)

	if taskCount == 0 {
		return
	}

	// Add tasks to queue
	added, err := p.queue.EnqueueMultiple(tasks)
	if err != nil {
		log.Printf("%s Warning: Some tasks were duplicates, added %d of %d tasks", p.logPrefix, added, taskCount)
	} else {
		log.Printf("%s Added %d task(s) to queue", p.logPrefix, added)
	}

	// Log current queue state
	queueSize := p.queue.Size()
	log.Printf("%s Queue size: %d task(s)", p.logPrefix, queueSize)
}

// GetMetrics returns the current metrics snapshot
func (p *PollingLoop) GetMetrics() MetricsSnapshot {
	return p.metrics.GetSnapshot()
}

// GetQueueSize returns the current queue size
func (p *PollingLoop) GetQueueSize() int {
	return p.queue.Size()
}

// GetQueue returns the task queue (for testing)
func (p *PollingLoop) GetQueue() *TaskQueue {
	return p.queue
}

// LogStatus logs the current status of the polling loop
func (p *PollingLoop) LogStatus() {
	snapshot := p.metrics.GetSnapshot()
	queueSize := p.queue.Size()

	log.Printf("%s === Status Report ===", p.logPrefix)
	log.Printf("%s Loop iterations: %d", p.logPrefix, snapshot.LoopCount)
	log.Printf("%s Total tasks fetched: %d", p.logPrefix, snapshot.TotalTasksFetched)
	log.Printf("%s Total heartbeats sent: %d", p.logPrefix, snapshot.TotalHeartbeatsSent)
	log.Printf("%s Total errors: %d (heartbeat: %d, task fetch: %d)",
		p.logPrefix, snapshot.TotalErrors, snapshot.HeartbeatErrors, snapshot.TaskFetchErrors)
	log.Printf("%s Last heartbeat: %s", p.logPrefix, snapshot.LastHeartbeatStatus)
	log.Printf("%s Last task fetch: %s", p.logPrefix, snapshot.LastTaskFetchStatus)
	log.Printf("%s Queue size: %d", p.logPrefix, queueSize)
	log.Printf("%s Average loop duration: %v", p.logPrefix, snapshot.AverageLoopDuration)

	if snapshot.LastError != "" {
		log.Printf("%s Last error: %s (at %v)", p.logPrefix, snapshot.LastError, snapshot.LastErrorTimestamp)
	}

	log.Printf("%s ==================", p.logPrefix)
}

// SetLogPrefix sets the log prefix (useful for testing)
func (p *PollingLoop) SetLogPrefix(prefix string) {
	p.logPrefix = prefix
}

// FormatMetrics returns a formatted string of current metrics
func (p *PollingLoop) FormatMetrics() string {
	snapshot := p.metrics.GetSnapshot()
	queueSize := p.queue.Size()

	return fmt.Sprintf(
		"Loops: %d | Tasks: %d | Heartbeats: %d | Errors: %d | Queue: %d | Avg Duration: %v",
		snapshot.LoopCount,
		snapshot.TotalTasksFetched,
		snapshot.TotalHeartbeatsSent,
		snapshot.TotalErrors,
		queueSize,
		snapshot.AverageLoopDuration,
	)
}
