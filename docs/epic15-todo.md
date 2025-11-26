# EPIC 15 — Implementation TODO

**Backend Complete:** EPICs 2-14 (558 tests passing)
**Ready For:** EPIC 15 Implementation

---

## Summary

EPIC 15 adds **per-agent concurrency control, quotas, and exponential backoff** to prevent system overload and ensure predictable task execution.

**Key Changes:**
- Agent configuration extended with concurrency/quota settings
- Agent executor respects concurrency limits locally
- Orchestrator scheduler aware of agent saturation
- Exponential backoff for failed tasks with retry budgets
- Backoff state signaling between agent and orchestrator
- Comprehensive metrics for concurrency and backoff

---

## Implementation Order (TDD-First)

### Phase 1: Configuration & Models (Story 1)
**Estimated Time:** 4 hours

1. **Add concurrency fields to Agent model**
   - `internal/store/models.go`
   - Fields: `MaxConcurrentTasks`, `MaxConcurrentBackups`, `MaxConcurrentChecks`, `MaxConcurrentPrunes`, `CPUQuotaPercent`, `BandwidthLimitMbps`
   - Create migration 006

2. **Add agent concurrency configuration**
   - `agent/concurrency.go` - ConcurrencyConfig type
   - Validation functions
   - Default values

3. **Update agent config loading**
   - `agent/config.go` - Add concurrency fields
   - Load from YAML + environment variables
   - Defaults if not set

4. **API for updating agent settings**
   - `internal/api/agents.go` - PATCH /agents/{id}/settings
   - Validation
   - Tests

**Tests to Write First:**
- [ ] `internal/store/models_test.go` - Agent concurrency fields
- [ ] `agent/concurrency_test.go` - Validation (valid/invalid ranges)
- [ ] `agent/config_test.go` - Config loading with defaults
- [ ] `internal/api/agents_test.go` - Settings update API

---

### Phase 2: Local Concurrency Control (Story 2)
**Estimated Time:** 6 hours

1. **Add concurrency tracking to executor**
   - `agent/executor.go` - Current task counters
   - Semaphore or channel-based limiting
   - Per-type limits (backup/check/prune)

2. **Add quota hints to Restic invocation**
   - `agent/executor.go` - Build Restic args with limits
   - `--limit-upload` for bandwidth
   - Process priority (nice) on Linux/macOS

3. **Queue blocking when at capacity**
   - Task queue blocks if limit reached
   - Tests with multiple concurrent tasks

**Tests to Write First:**
- [ ] `agent/executor_test.go` - Exceeding max blocks
- [ ] `agent/executor_test.go` - Back-to-back respects limits
- [ ] `agent/executor_test.go` - Quota flags in command
- [ ] `agent/executor_test.go` - Process priority applied

---

### Phase 3: Orchestrator Scheduling Awareness (Story 3)
**Estimated Time:** 4 hours

1. **Add agent load fields to heartbeat**
   - `internal/store/models.go` - Agent fields: `RunningTasks`, `QueueDepth`
   - Update via heartbeat

2. **Scheduler checks agent saturation**
   - `internal/scheduler/scheduler.go` - Query agent load
   - Skip task generation if saturated
   - Configurable threshold

3. **UI endpoint shows saturation state**
   - `internal/api/agents.go` - Include load in response
   - Status field: "saturated"

**Tests to Write First:**
- [ ] `internal/api/heartbeat_test.go` - Load fields update
- [ ] `internal/scheduler/scheduler_test.go` - Saturated agent skipped
- [ ] `internal/scheduler/scheduler_test.go` - Clearing saturation resumes
- [ ] `internal/api/agents_test.go` - Saturation in response

---

### Phase 4: Exponential Backoff (Story 4)
**Estimated Time:** 6 hours

1. **Add retry fields to Task model**
   - `internal/store/models.go` - `RetryCount`, `NextRetryAt`, `MaxRetries`
   - Migration 007

2. **Implement backoff calculation**
   - `agent/backoff.go` - ExponentialBackoff(retryCount) → duration
   - Cap at maxBackoffMinutes

3. **Agent executor respects backoff**
   - Skip tasks with NextRetryAt in future
   - Update retry count on failure
   - Mark as `failed_permanently` after max retries

4. **Persist backoff state**
   - Agent state file includes backoff timers
   - Restore on restart

**Tests to Write First:**
- [ ] `agent/backoff_test.go` - Retry schedule calculation
- [ ] `agent/backoff_test.go` - Backoff timer with mock clock
- [ ] `agent/executor_test.go` - Max attempts → permanent fail
- [ ] `agent/state_test.go` - Backoff persists across restart

---

### Phase 5: Retry Budget (Story 5)
**Estimated Time:** 3 hours

1. **Enforce retry budget in executor**
   - Check `task.RetryCount >= task.MaxRetries`
   - Fail immediately if exceeded

2. **Policy-level retry override**
   - `internal/store/models.go` - Policy field: `MaxTaskRetries`
   - Override task-level default

3. **API returns retry metadata**
   - `internal/api/tasks.go` - Include retry info in response
   - `internal/api/backup_runs.go` - Show failed-permanently

**Tests to Write First:**
- [ ] `agent/executor_test.go` - Exceeding budget → fail
- [ ] `internal/api/policies_test.go` - Policy retry override
- [ ] `internal/api/tasks_test.go` - Retry metadata in response

---

### Phase 6: Backoff Signaling (Story 6)
**Estimated Time:** 3 hours

1. **Add backoff fields to heartbeat**
   - `agent/heartbeat.go` - `BackoffUntil`, `PendingRetries`
   - Send in heartbeat payload

2. **Orchestrator reads backoff state**
   - `internal/api/heartbeat.go` - Store backoff fields
   - `internal/scheduler/scheduler.go` - Check before dispatching

3. **UI shows backoff status**
   - `internal/api/agents.go` - Include backoff in response
   - Badge: "Backoff active until X"

**Tests to Write First:**
- [ ] `agent/heartbeat_test.go` - Backoff fields sent
- [ ] `internal/api/heartbeat_test.go` - Backoff stored
- [ ] `internal/scheduler/scheduler_test.go` - No dispatch during backoff
- [ ] `internal/api/agents_test.go` - Backoff in status

---

### Phase 7: Metrics (Story 7)
**Estimated Time:** 4 hours

1. **Add concurrency metrics to agent**
   - `agent/metrics.go` - Extend LoopMetrics
   - Fields: `currentConcurrency`, `concurrencyLimit`, `backoffActive`, `retriesTotal`, `queueDepth`

2. **Update metrics on events**
   - Task start/end → update concurrency
   - Backoff enter/exit → update flag
   - Retry → increment counter

3. **Add logging for events**
   - "Concurrency limit reached"
   - "Entering backoff for X minutes"
   - "Retry exhausted for task Y"

**Tests to Write First:**
- [ ] `agent/metrics_test.go` - Metrics increment correctly
- [ ] `agent/metrics_test.go` - Backoff activates metric
- [ ] `agent/executor_test.go` - Queue depth tracked
- [ ] Log capture tests for events

---

## Database Migrations

### Migration 006: Agent Concurrency Settings
```sql
ALTER TABLE agents ADD COLUMN max_concurrent_tasks INTEGER DEFAULT 1;
ALTER TABLE agents ADD COLUMN max_concurrent_backups INTEGER DEFAULT 1;
ALTER TABLE agents ADD COLUMN max_concurrent_checks INTEGER DEFAULT 1;
ALTER TABLE agents ADD COLUMN max_concurrent_prunes INTEGER DEFAULT 1;
ALTER TABLE agents ADD COLUMN cpu_quota_percent INTEGER DEFAULT 50;
ALTER TABLE agents ADD COLUMN bandwidth_limit_mbps INTEGER;
ALTER TABLE agents ADD COLUMN running_tasks INTEGER DEFAULT 0;
ALTER TABLE agents ADD COLUMN queue_depth INTEGER DEFAULT 0;
```

### Migration 007: Task Retry Tracking
```sql
ALTER TABLE tasks ADD COLUMN retry_count INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN next_retry_at TIMESTAMP;
ALTER TABLE tasks ADD COLUMN max_retries INTEGER DEFAULT 5;
ALTER TABLE tasks ADD COLUMN failed_permanently BOOLEAN DEFAULT FALSE;
```

### Migration 008: Agent Backoff State
```sql
ALTER TABLE agents ADD COLUMN backoff_until TIMESTAMP;
ALTER TABLE agents ADD COLUMN pending_retries INTEGER DEFAULT 0;
```

### Migration 009: Policy Retry Overrides
```sql
ALTER TABLE policies ADD COLUMN max_task_retries INTEGER;
```

---

## Testing Strategy

### Mock Components Needed
1. **Mock Clock** - For time-dependent backoff tests
2. **Mock Restic Executor** - Verify command-line args
3. **Mock Orchestrator API** - Agent config polling
4. **Mock Process Manager** - Verify nice/priority calls

### Test Coverage Goals
- Concurrency: 100% (all limit scenarios)
- Backoff: 100% (all retry schedules)
- Retry budget: 100% (all threshold cases)
- Metrics: 100% (all counters)
- API: 100% (all endpoints)

### Performance Tests
- [ ] 100 concurrent task attempts → correct blocking
- [ ] 1000 retry calculations → sub-millisecond
- [ ] Backoff state persistence → <10ms

---

## Configuration Examples

### Agent Config (agent.yaml)
```yaml
orchestratorUrl: http://localhost:8080
pollingIntervalSeconds: 30
heartbeatIntervalSeconds: 60

# New concurrency settings
maxConcurrentTasks: 2
maxConcurrentBackups: 1
maxConcurrentChecks: 1
maxConcurrentPrunes: 1
cpuQuotaPercent: 50
bandwidthLimitMbps: 20

# New retry settings
maxRetries: 5
maxBackoffMinutes: 60
```

### Orchestrator Config
```yaml
scheduler:
  agentQueueThreshold: 10  # Skip if queue > 10
  respectAgentBackoff: true
```

---

## API Changes

### New Endpoints

#### Update Agent Settings
```http
PATCH /agents/{id}/settings
X-Tenant-ID: {uuid}

{
  "maxConcurrentTasks": 2,
  "cpuQuotaPercent": 70,
  "bandwidthLimitMbps": 50
}
```

#### Get Agent with Load
```http
GET /agents/{id}
Response includes:
{
  "runningTasks": 1,
  "queueDepth": 3,
  "backoffUntil": "2025-11-26T10:30:00Z",
  "pendingRetries": 2,
  "saturated": false
}
```

---

## Success Criteria

- [ ] All 7 user stories implemented
- [ ] 50+ new tests passing
- [ ] Total test count: 608+
- [ ] No regressions in existing 558 tests
- [ ] Documentation updated (open-tasks.md, ui-preparation.md)
- [ ] Migrations tested with SQLite and PostgreSQL
- [ ] Agent can handle 10+ concurrent policies without overload
- [ ] Orchestrator respects agent capacity limits
- [ ] Failed tasks backoff exponentially up to 1 hour
- [ ] Metrics visible in agent diagnostics

---

## Timeline

**Total Estimated Time:** 30 hours (4 days)

| Phase | Story | Time | Tests |
|-------|-------|------|-------|
| 1 | Configuration | 4h | 8 |
| 2 | Local Concurrency | 6h | 12 |
| 3 | Scheduler Awareness | 4h | 8 |
| 4 | Exponential Backoff | 6h | 12 |
| 5 | Retry Budget | 3h | 6 |
| 6 | Backoff Signaling | 3h | 8 |
| 7 | Metrics | 4h | 8 |
| **Total** | | **30h** | **62** |

---

## Ready to Start?

All prerequisites complete:
- ✅ EPICs 2-14 implemented (558 tests passing)
- ✅ Agent polling loop functional
- ✅ Scheduler generating tasks
- ✅ Task execution working
- ✅ Database migrations infrastructure ready

**Next:** Begin Phase 1 (Configuration & Models) with TDD approach.
