# EPIC 4: API for Agent Registration - Status

## Overview
Implementation of API endpoints that allow backup agents to identify themselves to the orchestrator.

## Completion Status

### ✅ User Story 1: Define Agent Registration API Contract
**Status**: COMPLETED

**Deliverables**:
- `docs/api/agent-registration.md` - Complete API specification
  - Request schema with validation rules
  - Response schema with all required fields
  - HTTP status codes (200, 201, 400, 401, 500)
  - 4 detailed examples (new registration, re-registration, validation error, auth error)
  - Security considerations and integration points
- Updated `docs/architecture.md` with API Documentation section
  - Added Agent APIs overview
  - Linked to detailed agent-registration.md

**Test Coverage**: N/A (documentation)

---

### ✅ User Story 2: Implement Agent Registration Endpoint
**Status**: COMPLETED

**Description**: As a backup agent, I want to register myself with the orchestrator so that it knows I exist and can assign backup policies to me.

**Deliverables**:
- `internal/api/agents.go` - Handler implementation
  - `AgentRegisterRequest` struct with all required fields
  - `AgentRegisterResponse` struct matching documented schema
  - `handleAgentRegister()` with full registration logic
  - `validateAgentRequest()` for field validation
  - Error handling with proper HTTP status codes
  - Idempotent operation (hostname-based deduplication)
  
- `internal/api/api.go` - Router integration
  - Added route: `POST /agents/register`
  - Updated `authMiddleware` to protect `/agents/*` routes
  
- `internal/api/agents_test.go` - Comprehensive TDD test suite
  - `TestAgentRegistrationRouterMapping` - Route existence
  - `TestAgentRegistrationValidation` - 7 validation test cases
    - Missing hostname
    - Empty hostname  
    - Missing os
    - Missing arch
    - Missing version
    - Hostname too long (>255 chars)
    - Malformed JSON
  - `TestAgentRegistrationDBWrite` - Database persistence
    - Creates new agent with correct fields
    - Updates existing agent on duplicate hostname
    - Verifies idempotency
  - `TestAgentRegistrationResponseSchema` - Response format validation
    - All required fields present
    - Valid UUID for agentId
    - RFC3339 timestamp format
  - `TestAgentRegistrationAuth` - Authentication
    - No auth header (401)
    - Invalid token (401)
    - Valid token (201/200)

**Test Results**: All tests passing ✅
```
=== RUN   TestAgentRegistrationRouterMapping
--- PASS: TestAgentRegistrationRouterMapping (0.00s)
=== RUN   TestAgentRegistrationValidation
--- PASS: TestAgentRegistrationValidation (0.00s)
=== RUN   TestAgentRegistrationDBWrite
--- PASS: TestAgentRegistrationDBWrite (0.00s)
=== RUN   TestAgentRegistrationResponseSchema
--- PASS: TestAgentRegistrationResponseSchema (0.00s)
=== RUN   TestAgentRegistrationAuth
--- PASS: TestAgentRegistrationAuth (0.00s)
PASS
ok      github.com/example/restic-monitor/internal/api  0.190s
```

**Implementation Details**:
- Uses existing authentication mechanism (Bearer token or Basic Auth)
- Creates new agent with status="online"
- Updates existing agent on duplicate hostname (prevents duplicates)
- Sets `last_seen_at` timestamp on registration
- Stores optional metadata as JSONB
- Returns 201 Created for new agents
- Returns 200 OK for updated agents
- Returns 400 Bad Request for validation errors
- Returns 401 Unauthorized for auth failures
- Returns 500 Internal Server Error for database errors

---

### ✅ User Story 3: Agent Metadata Auto-Update
**Status**: COMPLETED (implemented in User Story 2)

**Description**: As an orchestrator, I want to automatically update agent metadata when an agent re-registers so that I always have current information.

**Implementation**: Built into `handleAgentRegister()` function
- Detects existing agent by hostname
- Updates OS, Arch, Version, Metadata, LastSeenAt on re-registration
- Maintains same agent ID for idempotency
- Sets status to "online"

**Test Coverage**: Covered by `TestAgentRegistrationDBWrite/Updates_existing_agent_on_duplicate_hostname`

---

### ✅ User Story 4: Structured Registration Response
**Status**: COMPLETED (implemented in User Story 2)

**Description**: As a backup agent, I want to receive a structured response containing my agent ID and registration status so I can use this ID in future API calls.

**Implementation**: `AgentRegisterResponse` struct
- `agentId` (UUID string)
- `hostname` (string)
- `registeredAt` (RFC3339 timestamp)
- `updatedAt` (RFC3339 timestamp)
- `message` (string)

**Test Coverage**: Covered by `TestAgentRegistrationResponseSchema`

---

### ⏳ User Story 5: Logging and Metrics
**Status**: PARTIALLY COMPLETED

**Description**: As an operator, I want all agent registrations to be logged with appropriate detail so that I can audit agent activity.

**Current Implementation**:
- Log statements for new registrations: `log.Printf("Registered new agent: %s (ID: %s)", ...)`
- Log statements for updates: `log.Printf("Updated agent: %s (ID: %s)", ...)`
- Error logging: `log.Printf("Failed to create/update agent %s: %v", ...)`

**Pending**:
- Structured logging (consider using `slog` or similar)
- Metrics integration (consider Prometheus metrics)
- Audit trail table (for compliance/debugging)

---

## API Endpoint Summary

### POST /agents/register

**Authentication**: Required (Bearer token or Basic Auth)

**Request Body**:
```json
{
  "hostname": "web-server-01.example.com",
  "os": "linux",
  "arch": "amd64",
  "version": "1.0.0",
  "ip": "192.168.1.100",
  "metadata": {
    "restic_version": "0.16.2",
    "cpu_cores": 8
  }
}
```

**Success Response (201 Created)** - New agent:
```json
{
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "hostname": "web-server-01.example.com",
  "registeredAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z",
  "message": "Agent registered successfully"
}
```

**Success Response (200 OK)** - Updated agent:
```json
{
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "hostname": "web-server-01.example.com",
  "registeredAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-16T14:20:00Z",
  "message": "Agent metadata updated"
}
```

**Error Response (400 Bad Request)**:
```json
{
  "error": "Validation failed",
  "details": "hostname is required and cannot be empty"
}
```

**Error Response (401 Unauthorized)**: Plain text "Unauthorized"

---

## Test Coverage Summary

| Test Suite | Test Cases | Status |
|------------|------------|--------|
| Router Mapping | 1 | ✅ PASS |
| Validation | 7 | ✅ PASS |
| DB Write | 2 | ✅ PASS |
| Response Schema | 1 | ✅ PASS |
| Authentication | 3 | ✅ PASS |
| **Total** | **14** | **✅ ALL PASS** |

---

## Integration Points

1. **Database**: Uses `store.Agent` model with GORM
2. **Authentication**: Integrates with existing `authMiddleware`
3. **Router**: Added to main HTTP ServeMux in `internal/api/api.go`
4. **Migrations**: Compatible with v1 schema from EPIC 3

---

## Next Steps (Future Enhancements)

1. **Agent Heartbeat API** (Future EPIC)
   - Endpoint for agents to send periodic heartbeats
   - Update `last_seen_at` timestamp
   - Detect stale/offline agents

2. **Agent Task Polling API** (Future EPIC)
   - Endpoint for agents to poll for backup tasks
   - Return assigned backup policies
   - Long polling or WebSocket support

3. **Enhanced Logging**
   - Migrate to structured logging (`slog`)
   - Add log levels (debug, info, warn, error)
   - Include correlation IDs for request tracing

4. **Metrics Integration**
   - Prometheus metrics for registrations
   - Counters: total_registrations, new_agents, updated_agents
   - Gauges: active_agents, agents_by_status

5. **Audit Trail**
   - Create `agent_audit_log` table
   - Record all registration/update events
   - Include timestamp, agent ID, changes, requester IP

---

## Files Modified/Created

### Created
- `docs/api/agent-registration.md` (400+ lines)
- `internal/api/agents.go` (200 lines)
- `internal/api/agents_test.go` (280 lines)
- `docs/epic4-status.md` (this file)

### Modified
- `docs/architecture.md` (added API Documentation section)
- `internal/api/api.go` (added route and auth middleware update)

---

## Dependencies

- `github.com/google/uuid` v1.6.0 (for agent IDs)
- `github.com/stretchr/testify` v1.11.1 (for testing)
- Standard library: `net/http`, `encoding/json`, `time`, `log`

---

## Conclusion

EPIC 4 User Stories 1-4 are **COMPLETED** with full TDD test coverage. User Story 5 (Logging and Metrics) has basic implementation but can be enhanced in future iterations. All tests pass, and the implementation follows the documented API contract.

The agent registration endpoint is production-ready and can be used by backup agents to register themselves with the orchestrator.
