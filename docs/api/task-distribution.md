# Task Distribution API

**Version:** 1.0  
**Last Updated:** November 25, 2025  
**Status:** Active

---

## Overview

The Task Distribution API allows agents to retrieve pending tasks from the orchestrator. This is the core of the pull-based orchestration system where agents periodically poll for work.

---

## Endpoint: Get Agent Tasks

### Request

```
GET /agents/{agentId}/tasks
```

**Path Parameters:**
- `agentId` (string, required): UUID of the agent requesting tasks

**Headers:**
- `Authorization: Bearer {token}` (required): Agent authentication token

**Query Parameters:**
- `limit` (integer, optional): Maximum number of tasks to return (default: 10, max: 100)

### Response

**Success (200 OK)**

Returns an array of pending tasks for the agent.

```json
{
  "tasks": [
    {
      "taskId": "550e8400-e29b-41d4-a716-446655440000",
      "policyId": "660e8400-e29b-41d4-a716-446655440001",
      "taskType": "backup",
      "repository": "s3:my-bucket/restic-repo",
      "includePaths": ["/home", "/etc"],
      "excludePaths": ["/home/*/cache", "*.tmp"],
      "retention": {
        "keepLast": 7,
        "keepDaily": 14,
        "keepWeekly": 8,
        "keepMonthly": 12,
        "keepYearly": 3
      },
      "executionParams": {
        "parallelism": 4,
        "bandwidthLimitKbps": 10000,
        "timeoutSeconds": 3600
      },
      "createdAt": "2025-11-25T10:30:00Z",
      "scheduledFor": "2025-11-25T11:00:00Z"
    }
  ],
  "count": 1
}
```

**No Tasks (204 No Content)**

When no pending tasks are available for the agent.

**Error Responses**

- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: Agent ID not found
- `500 Internal Server Error`: Server-side error

---

## Task Schema

### Task Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `taskId` | string (UUID) | Yes | Unique identifier for the task |
| `policyId` | string (UUID) | Yes | Policy that generated this task |
| `taskType` | enum | Yes | Type of task: `backup`, `check`, or `prune` |
| `repository` | string | Yes | Repository location (e.g., `s3:bucket/path`) |
| `includePaths` | array[string] | No | Paths to include in backup |
| `excludePaths` | array[string] | No | Paths to exclude from backup |
| `retention` | object | No | Retention policy (for prune tasks) |
| `executionParams` | object | No | Execution parameters |
| `createdAt` | string (ISO 8601) | Yes | When the task was created |
| `scheduledFor` | string (ISO 8601) | No | When the task should be executed |

### Task Type Enum

- `backup`: Create a new backup snapshot
- `check`: Verify repository integrity
- `prune`: Apply retention policy and remove old snapshots

### Retention Object

Used for `prune` tasks to specify which snapshots to keep.

| Field | Type | Description |
|-------|------|-------------|
| `keepLast` | integer | Keep last N snapshots |
| `keepDaily` | integer | Keep last N daily snapshots |
| `keepWeekly` | integer | Keep last N weekly snapshots |
| `keepMonthly` | integer | Keep last N monthly snapshots |
| `keepYearly` | integer | Keep last N yearly snapshots |

### Execution Parameters Object

Optional parameters to control task execution.

| Field | Type | Description |
|-------|------|-------------|
| `parallelism` | integer | Number of parallel operations (default: 4) |
| `bandwidthLimitKbps` | integer | Bandwidth limit in Kbps (0 = unlimited) |
| `timeoutSeconds` | integer | Task timeout in seconds (default: 3600) |

---

## Task Acknowledgment (Optional)

### Request

```
POST /agents/{agentId}/tasks/{taskId}/ack
```

**Path Parameters:**
- `agentId` (string, required): UUID of the agent
- `taskId` (string, required): UUID of the task to acknowledge

**Headers:**
- `Authorization: Bearer {token}` (required): Agent authentication token

### Response

**Success (200 OK)**

```json
{
  "status": "acknowledged",
  "taskId": "550e8400-e29b-41d4-a716-446655440000",
  "acknowledgedAt": "2025-11-25T11:00:05Z"
}
```

**Error Responses**

- `401 Unauthorized`: Invalid or missing authentication token
- `404 Not Found`: Agent or task not found
- `409 Conflict`: Task already acknowledged or completed
- `500 Internal Server Error`: Server-side error

---

## Examples

### Example 1: Backup Task

```json
{
  "tasks": [
    {
      "taskId": "550e8400-e29b-41d4-a716-446655440000",
      "policyId": "660e8400-e29b-41d4-a716-446655440001",
      "taskType": "backup",
      "repository": "s3:my-bucket/restic-repo",
      "includePaths": ["/home/user/documents", "/etc"],
      "excludePaths": ["*.tmp", "/home/user/.cache"],
      "executionParams": {
        "parallelism": 4,
        "bandwidthLimitKbps": 5000,
        "timeoutSeconds": 7200
      },
      "createdAt": "2025-11-25T10:00:00Z",
      "scheduledFor": "2025-11-25T11:00:00Z"
    }
  ],
  "count": 1
}
```

### Example 2: Check Task

```json
{
  "tasks": [
    {
      "taskId": "550e8400-e29b-41d4-a716-446655440002",
      "policyId": "660e8400-e29b-41d4-a716-446655440001",
      "taskType": "check",
      "repository": "s3:my-bucket/restic-repo",
      "executionParams": {
        "timeoutSeconds": 1800
      },
      "createdAt": "2025-11-25T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Example 3: Prune Task

```json
{
  "tasks": [
    {
      "taskId": "550e8400-e29b-41d4-a716-446655440003",
      "policyId": "660e8400-e29b-41d4-a716-446655440001",
      "taskType": "prune",
      "repository": "s3:my-bucket/restic-repo",
      "retention": {
        "keepLast": 7,
        "keepDaily": 14,
        "keepWeekly": 8,
        "keepMonthly": 12,
        "keepYearly": 3
      },
      "createdAt": "2025-11-25T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Example 4: Empty Response

When no tasks are pending:

```
HTTP/1.1 204 No Content
```

---

## Validation Rules

### Required Fields
- `taskId` must be a valid UUID
- `policyId` must be a valid UUID
- `taskType` must be one of: `backup`, `check`, `prune`
- `repository` must not be empty
- `createdAt` must be a valid ISO 8601 timestamp

### Optional Fields
- `includePaths` and `excludePaths` are optional for backup tasks
- `retention` is required only for prune tasks
- `executionParams` are optional with sensible defaults
- `scheduledFor` is optional (defaults to immediate execution)

### Constraints
- `limit` query parameter: 1-100 (default: 10)
- Maximum tasks per response: 100
- Task arrays may be empty (204 response)

---

## State Management

### Task States

Tasks progress through the following states:

1. **pending**: Task created by scheduler, waiting to be fetched
2. **assigned**: Task fetched by agent (optional state)
3. **in-progress**: Task acknowledged by agent
4. **completed**: Task executed successfully
5. **failed**: Task execution failed

### State Transitions

```
pending → assigned → in-progress → completed
                                 → failed
```

### Idempotency

- Fetching tasks multiple times returns the same pending tasks
- Acknowledging a task twice returns 200 OK (idempotent)
- Completed or failed tasks are not returned in subsequent fetches

---

## Security

### Authentication

All requests require a valid Bearer token in the Authorization header. The token is provided during agent registration.

### Authorization

Agents can only fetch tasks assigned to them. Requests for other agents' tasks return 404 Not Found.

---

## Rate Limiting

Agents should poll for tasks at the configured `pollingIntervalSeconds` (default: 30 seconds). The API does not enforce rate limiting but excessive polling may result in throttling.

---

## Metrics & Logging

### Logged Events

- Task fetch request received
- Tasks returned to agent
- Task acknowledgment received
- Errors during task retrieval

### Metrics

- `tasks_fetched_total`: Counter of task fetch requests
- `tasks_assigned_total`: Counter of tasks assigned to agents
- `tasks_acknowledged_total`: Counter of acknowledged tasks
- `tasks_pending`: Gauge of pending tasks in the system

---

## Versioning

This API follows semantic versioning. Breaking changes will increment the major version number.

**Current Version:** 1.0

---

## Related Documentation

- [Agent Registration API](./agent-registration.md)
- [Agent Heartbeat API](./agent-heartbeat.md)
- [Policy Schema](../policies/policy-schema.md)
