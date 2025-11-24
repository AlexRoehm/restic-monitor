# Agent-Policy Assignment Specification

## Overview

This document defines how agents are linked to backup policies in the restic-monitor system. The relationship enables central orchestration: operators assign policies to agents, and agents fetch their assigned policies to execute backup, prune, and check operations.

## Relationship Model

### Many-to-Many Relationship

* **Agents can have multiple policies** — A single agent may execute different backup schedules for different data sets (e.g., hourly for critical data, daily for bulk storage)
* **Policies can be assigned to multiple agents** — A single policy can be reused across many machines (e.g., "Standard Workstation Backup" applied to all desktop machines)

This many-to-many relationship is implemented via a **join table**.

## Database Schema

### Join Table: `agent_policies`

```sql
CREATE TABLE agent_policies (
    agent_id  UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, policy_id)
);
```

### Schema Properties

* **Composite Primary Key**: `(agent_id, policy_id)` ensures no duplicate assignments
* **Cascade Deletes**: 
  * When an agent is deleted → all its policy assignments are removed
  * When a policy is deleted → all assignments to agents are removed
* **Foreign Key Constraints**: Enforces referential integrity (cannot assign non-existent policy or agent)

### GORM Model

```go
type AgentPolicy struct {
    AgentID  uuid.UUID `gorm:"primaryKey;type:uuid"`
    PolicyID uuid.UUID `gorm:"primaryKey;type:uuid"`
    Agent    Agent     `gorm:"constraint:OnDelete:CASCADE;"`
    Policy   Policy    `gorm:"constraint:OnDelete:CASCADE;"`
}
```

## API Endpoints

### 1. Assign Policy to Agent

**Endpoint**: `POST /agents/{agentId}/policies/{policyId}`

**Description**: Creates a new assignment linking a policy to an agent.

**Path Parameters**:
* `agentId` (UUID) — Agent identifier
* `policyId` (UUID) — Policy identifier

**Request**: Empty body

**Success Response** (201 Created):
```json
{
  "status": "assigned",
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "policyId": "660e8400-e29b-41d4-a716-446655440001"
}
```

**Error Responses**:
* `400 Bad Request` — Invalid UUID format
* `404 Not Found` — Agent or policy does not exist
* `409 Conflict` — Assignment already exists

**Example**:
```bash
curl -X POST http://localhost:8080/agents/550e8400-e29b-41d4-a716-446655440000/policies/660e8400-e29b-41d4-a716-446655440001
```

---

### 2. Remove Policy from Agent

**Endpoint**: `DELETE /agents/{agentId}/policies/{policyId}`

**Description**: Removes an existing assignment between a policy and an agent.

**Path Parameters**:
* `agentId` (UUID) — Agent identifier
* `policyId` (UUID) — Policy identifier

**Request**: Empty body

**Success Response** (200 OK):
```json
{
  "status": "removed"
}
```

**Error Responses**:
* `400 Bad Request` — Invalid UUID format
* `404 Not Found` — Assignment does not exist

**Example**:
```bash
curl -X DELETE http://localhost:8080/agents/550e8400-e29b-41d4-a716-446655440000/policies/660e8400-e29b-41d4-a716-446655440001
```

---

### 3. List Policies Assigned to Agent

**Endpoint**: `GET /agents/{agentId}/policies`

**Description**: Returns all policies assigned to a specific agent in agent-friendly format (excludes orchestrator metadata).

**Path Parameters**:
* `agentId` (UUID) — Agent identifier

**Success Response** (200 OK):
```json
{
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "policies": [
    {
      "name": "daily-home-backup",
      "description": "Daily backup of home directory",
      "schedule": "0 3 * * *",
      "includePaths": ["/home/user"],
      "excludePaths": ["/home/user/.cache"],
      "repository": {
        "type": "s3",
        "bucket": "my-backups",
        "endpoint": "s3.amazonaws.com",
        "region": "us-east-1"
      },
      "retention": {
        "keepLast": 7,
        "keepDaily": 30,
        "keepWeekly": 12,
        "keepMonthly": 12
      },
      "performance": {
        "bandwidthLimit": 10000,
        "parallelFiles": 4
      }
    }
  ]
}
```

**Error Responses**:
* `400 Bad Request` — Invalid UUID format
* `404 Not Found` — Agent does not exist

**Notes**:
* Empty array returned if agent has no assigned policies
* Only **enabled** policies are included
* Policies sorted alphabetically by name
* Response format matches Epic 6 agent serialization (excludes: `id`, `tenantId`, `enabled`, `createdAt`, `updatedAt`)

---

### 4. List Agents Assigned to Policy (Optional)

**Endpoint**: `GET /policies/{policyId}/agents`

**Description**: Returns all agents to which a specific policy is assigned. Useful for impact analysis and coverage reporting.

**Path Parameters**:
* `policyId` (UUID) — Policy identifier

**Success Response** (200 OK):
```json
{
  "policyId": "660e8400-e29b-41d4-a716-446655440001",
  "policyName": "daily-home-backup",
  "agents": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "hostname": "workstation-01",
      "lastSeenAt": "2025-11-24T14:30:00Z",
      "status": "online"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "hostname": "workstation-02",
      "lastSeenAt": "2025-11-24T14:28:00Z",
      "status": "online"
    }
  ]
}
```

**Error Responses**:
* `400 Bad Request` — Invalid UUID format
* `404 Not Found` — Policy does not exist

**Notes**:
* Empty array returned if policy not assigned to any agents
* Agent status determined by heartbeat (online if seen within 5 minutes)
* Agents sorted alphabetically by hostname

## Business Rules

### Assignment Rules

1. **Uniqueness**: An agent can be assigned the same policy only once (enforced by primary key)
2. **Existence**: Both agent and policy must exist in their respective tables before assignment
3. **Tenant Isolation**: Agents can only be assigned policies from the same tenant (enforced by application logic)
4. **Enabled Status**: Only enabled policies are returned to agents (disabled policies exist in DB but not sent to agent)

### Deletion Behavior

1. **Agent Deletion**: When an agent is deleted, all its policy assignments are automatically removed (CASCADE)
2. **Policy Deletion**: When a policy is deleted, all assignments to agents are automatically removed (CASCADE)
3. **Assignment Deletion**: Removing an assignment does not delete the agent or policy

### Query Efficiency

* **Agent → Policies**: Single JOIN query fetching all policies for an agent
* **Policy → Agents**: Single JOIN query fetching all agents for a policy
* Indexes on `agent_id` and `policy_id` ensure fast lookups

## Logging & Metrics

### Log Events

All assignment operations generate structured log entries:

* **INFO**: Policy assigned — `"Assigned policy 'daily-backup' (ID: 660e...) to agent 'workstation-01' (ID: 550e...)"`
* **INFO**: Policy removed — `"Removed policy 'daily-backup' (ID: 660e...) from agent 'workstation-01' (ID: 550e...)"`
* **WARN**: Assignment validation failure — `"Cannot assign policy 660e... to agent 550e...: policy not found"`
* **DEBUG**: Assignment lookup — `"Agent 550e... has 3 assigned policies"`

### Prometheus Metrics

* `policy_assign_total` (counter) — Total number of successful policy assignments
* `policy_unassign_total` (counter) — Total number of successful policy removals
* `agent_policy_assignments` (gauge) — Current total number of active assignments

## Use Cases

### 1. Standard Workstation Backup

**Scenario**: Apply same backup policy to 50 workstations

```bash
# Create policy once
POST /policies
{
  "name": "standard-workstation",
  "schedule": "0 2 * * *",
  ...
}

# Assign to each agent
for agent in $(list_agents); do
  POST /agents/$agent/policies/$policy_id
done
```

### 2. Multi-Tier Backup Strategy

**Scenario**: Agent backs up critical data hourly, bulk data daily

```bash
# Assign hourly policy for /var/www
POST /agents/$agent/policies/$hourly_policy

# Assign daily policy for /home
POST /agents/$agent/policies/$daily_policy
```

### 3. Policy Impact Analysis

**Scenario**: Check which agents will be affected by policy change

```bash
# List all agents using policy before modification
GET /policies/$policy_id/agents
```

### 4. Agent Configuration Audit

**Scenario**: Verify agent's backup configuration

```bash
# List all policies assigned to specific agent
GET /agents/$agent_id/policies
```

## Implementation Notes

### Testing Requirements (TDD)

Each API endpoint must have:

1. **Happy path tests** — Valid assignment/removal/listing
2. **Validation tests** — Invalid UUIDs, nonexistent resources
3. **Duplicate prevention tests** — Assigning same policy twice
4. **Cascade tests** — Deletion propagation
5. **Response schema tests** — JSON structure validation
6. **Edge case tests** — Empty lists, boundary conditions

### Performance Considerations

* Join queries optimized with indexes
* Agent policy fetch should complete in <50ms for typical loads
* Batch assignment operations not currently supported (future enhancement)

### Security Considerations

* All endpoints require authentication
* Tenant isolation enforced at application layer
* Agents can only read their own assignments (via agent authentication)
* Only operators can modify assignments (via operator authentication)

## Future Enhancements

* **Assignment Metadata**: Add `assigned_at` timestamp, `assigned_by` operator tracking
* **Assignment Priority**: Allow ordering when multiple policies conflict
* **Assignment Groups**: Bulk assign policies to agent groups (e.g., all agents with tag "production")
* **Assignment Scheduling**: Temporary assignments (e.g., one-time backup policy for migration)
* **Assignment Audit Trail**: Track history of assignments over time
