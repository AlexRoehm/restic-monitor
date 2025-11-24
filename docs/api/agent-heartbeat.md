# Agent Heartbeat API

## Overview

The Agent Heartbeat API allows backup agents to periodically notify the orchestrator of their status, health, and availability. This enables real-time monitoring, automatic online/offline detection, and informed scheduling decisions.

## Table of Contents

1. [Endpoint](#endpoint)
2. [Authentication](#authentication)
3. [Request Schema](#request-schema)
4. [Response Schema](#response-schema)
5. [Behavior](#behavior)
6. [Examples](#examples)
7. [Status Codes](#status-codes)
8. [Error Handling](#error-handling)
9. [Online/Offline Detection](#onlineoffline-detection)
10. [Integration Points](#integration-points)

---

## Endpoint

```
POST /agents/{agentId}/heartbeat
```

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `agentId` | UUID string | Yes | The unique identifier of the agent (returned during registration) |

---

## Authentication

This endpoint requires authentication using one of the following methods:

- **Bearer Token**: `Authorization: Bearer <token>`
- **Basic Auth**: `Authorization: Basic <base64(username:password)>`

The authentication mechanism is the same as used for agent registration.

---

## Request Schema

### Required Fields

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `version` | string | **Yes** | Max 50 chars | Agent software version (e.g., "1.2.3") |
| `os` | string | **Yes** | Max 50 chars | Operating system (e.g., "linux", "windows", "darwin") |

### Optional Fields

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `uptimeSeconds` | integer | No | >= 0 | System uptime in seconds |
| `disks` | array | No | Max 100 items | Disk usage information for all monitored mount points |
| `lastBackupStatus` | string | No | Enum: "success", "failure", "none", "running" | Status of the most recent backup operation |

### Disk Object Schema

Each item in the `disks` array must contain:

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `mountPath` | string | **Yes** | Max 255 chars | Mount point path (e.g., "/", "/data", "C:\\") |
| `freeBytes` | integer | **Yes** | >= 0 | Available space in bytes |
| `totalBytes` | integer | **Yes** | > 0 | Total capacity in bytes |

### Complete Request Structure

```json
{
  "version": "string",
  "os": "string",
  "uptimeSeconds": 0,
  "disks": [
    {
      "mountPath": "string",
      "freeBytes": 0,
      "totalBytes": 0
    }
  ],
  "lastBackupStatus": "success|failure|none|running"
}
```

---

## Response Schema

### Success Response

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Always "ok" for successful heartbeats |
| `nextTaskCheckAfterSeconds` | integer | Recommended interval (in seconds) before next heartbeat or task poll |

### Complete Response Structure

```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

---

## Behavior

### On Valid Heartbeat

The orchestrator performs the following actions:

1. **Validates Agent**: Confirms the agent ID exists in the database
2. **Updates Agent Record**:
   - `last_seen_at` → current timestamp
   - `version` → from request
   - `os` → from request
   - `status` → "online"
   - `last_backup_status` → from request (if provided)
   - `free_disk` → JSON array from `disks` field (if provided)
   - `uptime_seconds` → from request (if provided)
3. **Returns Success**: Provides `nextTaskCheckAfterSeconds` for polling guidance
4. **Logs Activity**: Records heartbeat reception at debug level
5. **Updates Metrics**: Increments `agent_heartbeats_total` counter

### Online/Offline Detection

- **Online**: Agent has sent a heartbeat within the configured threshold (default: 90 seconds)
- **Offline**: `last_seen_at` is older than the threshold
- **Status Calculation**: Automatically updated on each heartbeat and during periodic housekeeping

### Disk Information Storage

Disk data is stored as a JSON array in the `free_disk` column:

```json
[
  {"mountPath": "/", "freeBytes": 50000000000, "totalBytes": 100000000000},
  {"mountPath": "/data", "freeBytes": 200000000000, "totalBytes": 500000000000}
]
```

---

## Examples

### Example 1: Linux Server with Multiple Disks

**Request**:
```http
POST /agents/550e8400-e29b-41d4-a716-446655440000/heartbeat HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer abc123token
Content-Type: application/json

{
  "version": "1.2.3",
  "os": "linux",
  "uptimeSeconds": 864000,
  "disks": [
    {
      "mountPath": "/",
      "freeBytes": 50000000000,
      "totalBytes": 100000000000
    },
    {
      "mountPath": "/data",
      "freeBytes": 200000000000,
      "totalBytes": 500000000000
    }
  ],
  "lastBackupStatus": "success"
}
```

**Response** (200 OK):
```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

### Example 2: Windows Server

**Request**:
```http
POST /agents/7c9e6679-7425-40de-944b-e07fc1f90ae7/heartbeat HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer abc123token
Content-Type: application/json

{
  "version": "2.0.0",
  "os": "windows",
  "uptimeSeconds": 432000,
  "disks": [
    {
      "mountPath": "C:\\",
      "freeBytes": 75000000000,
      "totalBytes": 250000000000
    },
    {
      "mountPath": "D:\\",
      "freeBytes": 150000000000,
      "totalBytes": 1000000000000
    }
  ],
  "lastBackupStatus": "success"
}
```

**Response** (200 OK):
```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

### Example 3: macOS Desktop (Minimal Heartbeat)

**Request**:
```http
POST /agents/9f4e5cd7-8a12-4b3c-9d2e-1f5a6b7c8d9e/heartbeat HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer abc123token
Content-Type: application/json

{
  "version": "1.0.0",
  "os": "darwin"
}
```

**Response** (200 OK):
```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

### Example 4: Agent with Failed Backup

**Request**:
```http
POST /agents/550e8400-e29b-41d4-a716-446655440000/heartbeat HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer abc123token
Content-Type: application/json

{
  "version": "1.2.3",
  "os": "linux",
  "uptimeSeconds": 864500,
  "disks": [
    {
      "mountPath": "/",
      "freeBytes": 5000000000,
      "totalBytes": 100000000000
    }
  ],
  "lastBackupStatus": "failure"
}
```

**Response** (200 OK):
```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

---

## Status Codes

| Status Code | Meaning | Description |
|-------------|---------|-------------|
| **200 OK** | Success | Heartbeat received and processed successfully |
| **400 Bad Request** | Validation Error | Invalid request body, missing required fields, or malformed JSON |
| **401 Unauthorized** | Authentication Failed | Missing or invalid authentication credentials |
| **404 Not Found** | Agent Not Found | The specified agent ID does not exist in the database |
| **500 Internal Server Error** | Server Error | Database error or other internal failure |

---

## Error Handling

### Error Response Format

All errors return a JSON object with the following structure:

```json
{
  "error": "string",
  "details": "string"
}
```

### Example Error Responses

#### Missing Required Field (400)

**Request**:
```json
{
  "os": "linux"
}
```

**Response** (400 Bad Request):
```json
{
  "error": "Validation failed",
  "details": "version is required and cannot be empty"
}
```

#### Invalid Agent ID (404)

**Request**: Valid payload to non-existent agent

**Response** (404 Not Found):
```json
{
  "error": "Agent not found",
  "details": "No agent found with ID: 00000000-0000-0000-0000-000000000000"
}
```

#### Malformed JSON (400)

**Request**:
```
{invalid json}
```

**Response** (400 Bad Request):
```json
{
  "error": "Invalid request body",
  "details": "invalid character 'i' looking for beginning of object key string"
}
```

#### Invalid Disk Structure (400)

**Request**:
```json
{
  "version": "1.0.0",
  "os": "linux",
  "disks": [
    {
      "mountPath": "/",
      "freeBytes": -100
    }
  ]
}
```

**Response** (400 Bad Request):
```json
{
  "error": "Validation failed",
  "details": "disk freeBytes must be >= 0"
}
```

---

## Online/Offline Detection

### Configuration

The heartbeat threshold is configurable via the orchestrator configuration file:

```json
{
  "heartbeatTimeoutSeconds": 90
}
```

Default value: **90 seconds**

### Status Determination Logic

```
IF last_seen_at > NOW() - heartbeatTimeoutSeconds THEN
  status = "online"
ELSE
  status = "offline"
END IF
```

### Status Transitions

| Previous Status | Event | New Status | Logged |
|----------------|-------|------------|--------|
| offline | Heartbeat received | online | Yes (Info level) |
| online | Heartbeat received | online | No |
| online | Timeout exceeded | offline | Yes (Warn level) |

---

## Integration Points

### Database Updates

The following `agents` table columns are updated:

- `last_seen_at` (timestamp)
- `version` (string)
- `os` (string)
- `status` (enum: online, offline)
- `last_backup_status` (enum: success, failure, none, running)
- `free_disk` (JSONB array)
- `uptime_seconds` (integer)
- `updated_at` (timestamp, auto-updated)

### Metrics

The following Prometheus metrics are updated:

- `agent_heartbeats_total{agent_id, status}` - Counter of all heartbeats
- `agent_online_total` - Gauge of currently online agents
- `agent_heartbeat_duration_seconds` - Histogram of heartbeat processing time

### Logging

- **Debug**: All heartbeat contents (version, OS, uptime, disk count)
- **Info**: Status transitions (offline → online)
- **Warn**: Agents transitioning to offline
- **Error**: Database failures, validation errors

### Frontend Integration

The frontend can display:

- Real-time agent status (via GET /agents)
- Last seen timestamp
- Disk usage visualization
- Backup status indicators
- Uptime information

### Scheduler Integration

The backup scheduler uses:

- `status` field to avoid scheduling on offline agents
- `last_backup_status` to prioritize failed backups
- `free_disk` to ensure sufficient storage for backups

---

## Field Evolution

### Current Version (v1)

All fields documented above are part of the initial heartbeat API.

### Future Enhancements

Potential additions in future versions:

- `cpuUsagePercent` - Current CPU utilization
- `memoryUsedBytes` - RAM usage
- `networkInterfaces` - Network configuration
- `resticVersion` - Specific restic binary version
- `repositoryStatus` - Health of connected restic repositories
- `scheduledBackupCount` - Number of pending backup tasks

When adding new fields:
1. Add to this specification
2. Update request validation
3. Add database column or JSON field
4. Update TDD tests
5. Update metrics if needed

---

## Security Considerations

1. **Authentication Required**: All heartbeat requests must be authenticated
2. **Agent ID Validation**: Only accept heartbeats for existing agents
3. **Rate Limiting**: Consider implementing rate limits (e.g., max 1 heartbeat per agent per 10 seconds)
4. **Disk Path Validation**: Validate mount paths to prevent directory traversal attacks
5. **Data Size Limits**: Limit array sizes (max 100 disks) to prevent DoS
6. **JSON Bomb Protection**: Reject deeply nested or excessively large JSON payloads

---

## Performance Considerations

1. **Database Updates**: Use single UPDATE query per heartbeat
2. **Index Requirements**: Index on `last_seen_at` for status queries
3. **JSON Field**: Store disk data as JSON to avoid JOIN overhead
4. **Batch Processing**: Housekeeping task processes status updates in batches
5. **Caching**: Consider caching online/offline counts for large fleets (1000+ agents)

---

## Testing Requirements

All implementations must include TDD tests covering:

1. **Router Registration**: Endpoint is properly mapped
2. **Request Validation**: Required fields, data types, constraints
3. **Database Updates**: Correct fields updated with correct values
4. **404 Handling**: Non-existent agent IDs return 404
5. **Response Schema**: Output matches documented structure
6. **Online Detection**: Recent heartbeat marks agent online
7. **Offline Detection**: Stale heartbeat marks agent offline
8. **Disk Storage**: Disk array correctly serialized/deserialized
9. **Metrics**: Counters and gauges updated correctly
10. **Logging**: Appropriate log levels and content

---

## Changelog

### v1.0 (Current)
- Initial heartbeat API specification
- Support for version, OS, uptime, disks, last backup status
- Online/offline detection
- Metrics and logging integration
