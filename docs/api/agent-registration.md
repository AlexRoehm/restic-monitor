# Agent Registration API

## Overview

The Agent Registration API allows backup agents to identify themselves to the orchestrator and become managed machines. This is the first step in the agent lifecycle and enables the orchestrator to track agent metadata, assign policies, and schedule backup tasks.

## Authentication

All agent registration requests must use the existing authentication mechanism configured in the orchestrator:

- **Basic Auth**: Username/password via HTTP Basic Authentication header
- **Static Token**: Bearer token via `Authorization` header
- **Environment Token**: Token from `RESTIC_MONITOR_API_TOKEN` environment variable

The same authentication used for UI/API access applies to agent endpoints.

## Endpoint

### Register Agent

**Endpoint**: `POST /agents/register`

**Description**: Registers a new agent or updates an existing agent's metadata. The orchestrator uses hostname as the unique identifier for agents. If an agent with the same hostname already exists, its metadata is updated instead of creating a duplicate entry.

#### Request Schema

**Content-Type**: `application/json`

```json
{
  "hostname": "string (required)",
  "os": "string (required)",
  "arch": "string (required)",
  "version": "string (required)",
  "ip": "string (optional)",
  "metadata": {
    "key": "value (optional)"
  }
}
```

**Field Descriptions**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `hostname` | string | Yes | Fully qualified hostname or machine identifier. Used as unique key for agent identification. Maximum 255 characters. |
| `os` | string | Yes | Operating system name. Examples: `linux`, `windows`, `darwin`. Maximum 50 characters. |
| `arch` | string | Yes | CPU architecture. Examples: `amd64`, `arm64`, `386`. Maximum 50 characters. |
| `version` | string | Yes | Agent version string. Semantic versioning recommended (e.g., `1.2.3`). Maximum 50 characters. |
| `ip` | string | No | Agent's IP address or network identifier. Used for debugging and network diagnostics. |
| `metadata` | object | No | Custom metadata as key-value pairs. Can include restic version, installed plugins, system info, etc. Stored as JSON. |

**Validation Rules**:

- `hostname` must not be empty and must not exceed 255 characters
- `os` must not be empty and must not exceed 50 characters
- `arch` must not be empty and must not exceed 50 characters
- `version` must not be empty and must not exceed 50 characters
- `metadata` must be valid JSON object if provided

#### Response Schema

**Content-Type**: `application/json`

**Success Response (200 OK / 201 Created)**:

```json
{
  "agentId": "uuid",
  "hostname": "string",
  "registeredAt": "RFC3339 timestamp",
  "updatedAt": "RFC3339 timestamp",
  "message": "string"
}
```

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| `agentId` | string (UUID) | Unique identifier for the registered agent. Used in all subsequent API calls. |
| `hostname` | string | Echo of the registered hostname for confirmation. |
| `registeredAt` | string | RFC 3339 timestamp when the agent was first registered (created_at). |
| `updatedAt` | string | RFC 3339 timestamp when the agent record was last updated. |
| `message` | string | Human-readable status message. Examples: "Agent registered successfully", "Agent metadata updated". |

**HTTP Status Codes**:

| Status | Meaning |
|--------|---------|
| 200 OK | Agent already existed and metadata was updated |
| 201 Created | New agent was registered |
| 400 Bad Request | Invalid request body or missing required fields |
| 401 Unauthorized | Missing or invalid authentication credentials |
| 500 Internal Server Error | Database or server error |

**Error Response**:

```json
{
  "error": "string",
  "details": "string (optional)"
}
```

#### Examples

**Example 1: New Agent Registration**

Request:
```http
POST /agents/register HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer <token>
Content-Type: application/json

{
  "hostname": "web-server-01.example.com",
  "os": "linux",
  "arch": "amd64",
  "version": "1.0.0",
  "ip": "192.168.1.100",
  "metadata": {
    "restic_version": "0.16.2",
    "plugins": ["mysql", "postgres"],
    "cpu_cores": 8,
    "total_memory_gb": 32
  }
}
```

Response (201 Created):
```json
{
  "agentId": "a3f7c8e9-4d2a-4b5c-9e8f-7d6c5b4a3f2e",
  "hostname": "web-server-01.example.com",
  "registeredAt": "2025-11-24T12:00:00Z",
  "updatedAt": "2025-11-24T12:00:00Z",
  "message": "Agent registered successfully"
}
```

**Example 2: Agent Re-Registration (Update)**

Request:
```http
POST /agents/register HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer <token>
Content-Type: application/json

{
  "hostname": "web-server-01.example.com",
  "os": "linux",
  "arch": "amd64",
  "version": "1.1.0",
  "ip": "192.168.1.100"
}
```

Response (200 OK):
```json
{
  "agentId": "a3f7c8e9-4d2a-4b5c-9e8f-7d6c5b4a3f2e",
  "hostname": "web-server-01.example.com",
  "registeredAt": "2025-11-24T12:00:00Z",
  "updatedAt": "2025-11-24T14:30:00Z",
  "message": "Agent metadata updated"
}
```

**Example 3: Validation Error**

Request:
```http
POST /agents/register HTTP/1.1
Host: orchestrator.example.com
Authorization: Bearer <token>
Content-Type: application/json

{
  "hostname": "",
  "os": "linux"
}
```

Response (400 Bad Request):
```json
{
  "error": "Validation failed",
  "details": "hostname is required and cannot be empty; arch is required; version is required"
}
```

**Example 4: Authentication Error**

Request:
```http
POST /agents/register HTTP/1.1
Host: orchestrator.example.com
Content-Type: application/json

{
  "hostname": "web-server-01.example.com",
  "os": "linux",
  "arch": "amd64",
  "version": "1.0.0"
}
```

Response (401 Unauthorized):
```json
{
  "error": "Unauthorized",
  "details": "Missing or invalid authentication credentials"
}
```

## Behavior Details

### Idempotency

The registration endpoint is idempotent when called with the same hostname:

- First call: Creates new agent (201 Created)
- Subsequent calls: Updates existing agent metadata (200 OK)
- Same `agentId` returned in all cases

### Duplicate Hostname Handling

When an agent with the same hostname already exists:

1. The orchestrator locates the existing agent record by hostname
2. Updates the following fields:
   - `os`
   - `arch`
   - `version`
   - `ip` (if provided)
   - `metadata` (if provided)
   - `last_seen_at` (set to current time)
   - `status` (set to "online")
3. Does NOT create a duplicate entry
4. Returns the existing `agentId`

This behavior ensures machines can re-register after:
- Agent software upgrades
- OS reinstalls
- Network changes
- Service restarts

### Metadata Storage

The `metadata` field is stored as JSONB in PostgreSQL, allowing:

- Flexible schema-less storage
- Query capabilities on metadata fields
- Future extensibility without schema changes

Common metadata fields include:
- `restic_version`: Version of restic binary
- `plugins`: List of installed backup plugins
- `cpu_cores`: Number of CPU cores
- `total_memory_gb`: Total system memory
- `disk_space_gb`: Available disk space
- `backup_paths`: Default paths to backup

## Security Considerations

1. **Authentication Required**: All registration requests must be authenticated
2. **Hostname Validation**: Hostnames are validated to prevent SQL injection
3. **Rate Limiting**: Consider implementing rate limiting to prevent abuse
4. **Metadata Size**: Metadata field should have size limits (recommended: max 1MB)
5. **IP Validation**: IP addresses should be validated if provided

## Integration Points

### Database

- Writes to `agents` table created in EPIC 3
- Uses GORM models: `store.Agent`
- Tenant isolation via `tenant_id` field

### Future APIs

Agent registration enables:

- **Heartbeat API** (EPIC 5): Agents use `agentId` to send periodic health checks
- **Task Polling API** (EPIC 6): Agents use `agentId` to poll for backup tasks
- **Policy Assignment** (EPIC 7): Operators assign policies to registered agents
- **Backup Run Tracking** (EPIC 8): Track which agent executed which backup

## Testing

### Required Test Coverage

1. **API Contract Tests**: Validate request/response schemas
2. **Validation Tests**: Test all validation rules
3. **Duplicate Handling Tests**: Verify idempotency
4. **Authentication Tests**: Ensure auth is required
5. **Database Integration Tests**: Verify persistence
6. **Error Handling Tests**: Test all error scenarios

### Test Data

Example test fixtures are available in `internal/api/testdata/agent-registration/`.

## References

- [Architecture Documentation](../architecture.md#agent-components)
- [Database Schema](../architecture.md#database-schema)
- [Agent Model](../../internal/store/models.go)
- [Migration System](../architecture.md#database-migrations)

---

**Version**: 1.0.0  
**Last Updated**: 2025-11-24  
**Status**: Draft
