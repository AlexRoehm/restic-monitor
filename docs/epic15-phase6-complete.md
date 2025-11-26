# EPIC 15 Phase 6: Backoff Signaling - COMPLETE ✅

**Completion Date**: November 26, 2025  
**Tests Added**: 7 new tests (4 API + 3 migration tests)  
**Total Test Count**: 669 tests passing

## Overview

Phase 6 adds visibility into agent backoff state, allowing the orchestrator and operators to see which tasks are waiting in exponential backoff and when they'll retry next.

## Implementation Summary

### 1. Database Schema (Migration 009)

**File**: `internal/store/models.go`
- Added `TasksInBackoff *int` field to Agent model (default: 0)
- Added `EarliestRetryAt *time.Time` field to Agent model (indexed for queries)

**File**: `internal/store/migrations.go`
- Created `GetMigration009AddAgentBackoffState()` migration
- Uses AutoMigrate to add new columns to agents table

### 2. Backoff Status Endpoint

**File**: `internal/api/agent_backoff.go` (177 lines, NEW)

**Types**:
```go
type AgentBackoffResponse struct {
    AgentID         uuid.UUID          `json:"agent_id"`
    Hostname        string             `json:"hostname"`
    TasksInBackoff  int                `json:"tasks_in_backoff"`
    EarliestRetryAt *time.Time         `json:"earliest_retry_at,omitempty"`
    BackoffTasks    []BackoffTaskInfo  `json:"backoff_tasks"`
    LastUpdatedAt   time.Time          `json:"last_updated_at"`
}

type BackoffTaskInfo struct {
    TaskID        uuid.UUID  `json:"task_id"`
    TaskType      string     `json:"task_type"`
    RetryCount    int        `json:"retry_count"`
    MaxRetries    int        `json:"max_retries"`
    NextRetryAt   *time.Time `json:"next_retry_at"`
    ErrorCategory string     `json:"error_category,omitempty"`
}
```

**Endpoint**: `GET /agents/{id}/backoff-status`
- Returns agent backoff state with list of tasks in backoff
- Queries: `WHERE status='pending' AND next_retry_at > NOW()`
- Returns task details: type, retry count, max retries, next retry time, error category

**Functions**:
- `handleGetAgentBackoff()` - HTTP handler for backoff status endpoint
- `UpdateAgentBackoffState(agentID uuid.UUID)` - Calculates and updates agent backoff state
  - Counts tasks in backoff for the agent
  - Finds earliest retry time
  - Updates Agent.TasksInBackoff and Agent.EarliestRetryAt
  - Non-blocking (logs errors but doesn't fail)
- `safeIntValue()` - Helper for safe pointer dereferencing

### 3. Heartbeat Integration

**File**: `internal/api/heartbeat.go`
- Added `UpdateAgentBackoffState()` call in heartbeat handler
- Called after processing heartbeat, before sending response
- Non-blocking: logs errors but doesn't fail heartbeat
- Updates backoff state every ~30 seconds (heartbeat interval)

**File**: `internal/api/agents_list.go`
- Registered `/agents/{id}/backoff-status` route

## Test Coverage

### API Tests (4 tests)
**File**: `internal/api/agent_backoff_test.go`

1. `TestGetAgentBackoffStatus`
   - Creates agent with tasks in backoff (2 tasks with future retry times)
   - Creates task not in backoff (past retry time)
   - Verifies endpoint returns correct count (2), earliest retry time
   - Verifies task details in response

2. `TestUpdateAgentBackoffState`
   - Creates agent with 3 tasks in backoff
   - Calls UpdateAgentBackoffState()
   - Verifies agent.TasksInBackoff = 3
   - Verifies agent.EarliestRetryAt matches earliest task

3. `TestGetAgentBackoffStatusNoTasks`
   - Agent with no tasks
   - Verifies response: tasks_in_backoff = 0, earliest_retry_at = null

4. `TestGetAgentBackoffStatusAgentNotFound`
   - Invalid agent ID
   - Verifies 404 response

### Migration Tests (3 subtests)
**File**: `internal/store/migrations_test.go`

`TestMigration009AddAgentBackoffState`:
1. Create agent with default backoff state
   - Verifies tasks_in_backoff defaults to 0
   - Verifies earliest_retry_at defaults to NULL

2. Update agent backoff state
   - Creates agent, updates with backoff values
   - Verifies values persist correctly

3. Reset backoff state to zero
   - Creates agent with backoff state
   - Resets to 0 and NULL
   - Verifies reset successful

## Use Cases

### 1. Operator Dashboard
```bash
GET /agents/123/backoff-status
```
Response shows:
- How many tasks are waiting in backoff
- When the next retry will occur
- List of specific tasks with retry details
- Error categories for failed tasks

### 2. Load Balancing / Scheduling
- Orchestrator can query `agent.tasks_in_backoff` to avoid overloading agents
- Can schedule new tasks to agents with fewer backoff tasks
- Can predict when agent capacity will free up

### 3. Monitoring / Alerting
- Alert if an agent has too many tasks in backoff (e.g., > 10)
- Alert if earliest_retry_at is too far in the future (e.g., > 1 hour)
- Track backoff rate as a health metric

## Performance Considerations

1. **Indexed Field**: `earliest_retry_at` is indexed for efficient queries
2. **Cached State**: Backoff counts cached in Agent table, no need to count tasks on every query
3. **Update Frequency**: Updated every heartbeat (~30s), not on every task state change
4. **Non-blocking**: UpdateAgentBackoffState() logs errors but doesn't block heartbeat

## Example Response

```json
{
  "agent_id": "a2d9ccd9-0e33-4003-a492-57ed503ed513",
  "hostname": "backup-server-01",
  "tasks_in_backoff": 3,
  "earliest_retry_at": "2025-11-26T12:04:45Z",
  "backoff_tasks": [
    {
      "task_id": "task-1",
      "task_type": "backup",
      "retry_count": 1,
      "max_retries": 3,
      "next_retry_at": "2025-11-26T12:04:45Z",
      "error_category": "network"
    },
    {
      "task_id": "task-2",
      "task_type": "check",
      "retry_count": 2,
      "max_retries": 3,
      "next_retry_at": "2025-11-26T12:09:45Z"
    },
    {
      "task_id": "task-3",
      "task_type": "backup",
      "retry_count": 1,
      "max_retries": 3,
      "next_retry_at": "2025-11-26T12:14:45Z",
      "error_category": "resource"
    }
  ],
  "last_updated_at": "2025-11-26T11:59:45Z"
}
```

## Integration Points

- **Heartbeat Handler**: Updates backoff state every 30 seconds
- **Task Result Handler**: Changes task status, which affects backoff counts on next heartbeat
- **Backoff Calculation**: Uses `agent/backoff.go` logic for retry timing
- **Frontend**: Can display backoff status in agent detail view

## What's Next

Phase 7: Metrics & Observability
- Add Prometheus metrics for task execution, retries, backoff events
- Add structured logging for quota/backoff events
- Add performance metrics for task duration, queue times
- Estimated: 8-10 new tests

## Files Changed

### New Files
- `internal/api/agent_backoff.go` (177 lines)
- `internal/api/agent_backoff_test.go` (180 lines)

### Modified Files
- `internal/store/models.go` (added Agent.TasksInBackoff, Agent.EarliestRetryAt)
- `internal/store/migrations.go` (added GetMigration009)
- `internal/store/migrations_test.go` (added TestMigration009)
- `internal/api/heartbeat.go` (added UpdateAgentBackoffState call)
- `internal/api/agents_list.go` (added backoff-status route)

## Summary

Phase 6 successfully adds backoff visibility to the orchestrator:
- ✅ Migration 009 adds backoff tracking fields to Agent model
- ✅ GET /agents/{id}/backoff-status endpoint returns backoff state
- ✅ UpdateAgentBackoffState() calculates backoff metrics
- ✅ Heartbeat integration keeps backoff state current
- ✅ 7 new tests verify functionality
- ✅ All 669 tests passing

The orchestrator can now track which agents have tasks in backoff, when those tasks will retry, and what errors caused the backoff. This enables better scheduling decisions, operator visibility, and monitoring/alerting.
