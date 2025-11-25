# UI Development Preparation - Restic Monitor

**Date:** 2025-11-25
**Backend Status:** 558 tests passing, 12/13 EPICs complete

---

## Executive Summary

The backend orchestrator is **production-ready** for UI integration. All core APIs are implemented, tested, and functioning. This document provides a complete reference for frontend developers to build the UI.

---

## 1. Available APIs

### 1.1 Agent Management

#### List Agents
```http
GET /agents
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "agents": [
    {
      "id": "uuid",
      "hostname": "backup-server-1",
      "os": "linux",
      "arch": "amd64",
      "version": "1.0.0",
      "status": "online",
      "ip": "192.168.1.100",
      "lastHeartbeat": "2025-11-25T10:00:00Z",
      "registeredAt": "2025-11-20T09:00:00Z",
      "uptimeSeconds": 43200,
      "disks": [
        {
          "mountPath": "/data",
          "totalBytes": 1000000000000,
          "freeBytes": 500000000000,
          "usedPercent": 50.0
        }
      ]
    }
  ]
}
```

#### Get Agent Details
```http
GET /agents/{agentId}
X-Tenant-ID: {tenant-uuid}
```

#### Register Agent
```http
POST /agents/register
X-Tenant-ID: {tenant-uuid}
Content-Type: application/json

{
  "hostname": "backup-server-1",
  "os": "linux",
  "arch": "amd64",
  "version": "1.0.0",
  "ip": "192.168.1.100",
  "metadata": {}
}
```

#### Agent Heartbeat
```http
POST /agents/{agentId}/heartbeat
X-Tenant-ID: {tenant-uuid}

{
  "version": "1.0.0",
  "os": "linux",
  "uptimeSeconds": 43200,
  "lastBackupStatus": "success",
  "disks": [...]
}
```

---

### 1.2 Policy Management

#### List Policies
```http
GET /policies
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "policies": [
    {
      "id": "uuid",
      "name": "daily-backup",
      "description": "Daily backup at 2 AM",
      "schedule": "0 2 * * *",
      "checkSchedule": "0 6 * * *",
      "pruneSchedule": "0 3 * * 0",
      "repositoryUrl": "s3:s3.amazonaws.com/my-bucket",
      "repositoryType": "s3",
      "repositoryConfig": {
        "bucket": "my-bucket",
        "region": "us-east-1"
      },
      "includePaths": {
        "paths": ["/data", "/home"]
      },
      "excludePaths": {
        "patterns": ["*.tmp", "node_modules"]
      },
      "retentionRules": {
        "keepLast": 7,
        "keepDaily": 30,
        "keepWeekly": 52
      },
      "bandwidthLimitKBps": 10240,
      "parallelFiles": 4,
      "enabled": true,
      "createdAt": "2025-11-20T09:00:00Z",
      "updatedAt": "2025-11-20T09:00:00Z"
    }
  ]
}
```

#### Create Policy
```http
POST /policies
X-Tenant-ID: {tenant-uuid}
Content-Type: application/json

{
  "name": "daily-backup",
  "description": "Daily backup at 2 AM",
  "schedule": "0 2 * * *",
  "repositoryUrl": "s3:s3.amazonaws.com/my-bucket",
  "repositoryType": "s3",
  "repositoryConfig": {...},
  "includePaths": {"paths": ["/data"]},
  "retentionRules": {"keepLast": 7},
  "enabled": true
}
```

**Validation Rules:**
- Name: 3-100 chars, alphanumeric + hyphen/underscore
- Schedule: Valid cron (5 fields) or interval ("every 6h", "every 30m")
- Repository type: "s3", "rest-server", "filesystem", "sftp"
- Include paths: 1-100 absolute paths
- Retention: Positive integers for keep rules

#### Update Policy
```http
PUT /policies/{policyId}
X-Tenant-ID: {tenant-uuid}

{...updated fields...}
```

#### Delete Policy
```http
DELETE /policies/{policyId}
X-Tenant-ID: {tenant-uuid}
```

---

### 1.3 Policy-Agent Assignment

#### Get Policies for Agent
```http
GET /agents/{agentId}/policies
X-Tenant-ID: {tenant-uuid}
```

**Response:** Array of policies assigned to agent (excludes orchestrator metadata)

#### Get Agents for Policy
```http
GET /policies/{policyId}/agents
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "agents": [
    {
      "id": "uuid",
      "hostname": "backup-server-1",
      "status": "online",
      "lastHeartbeat": "2025-11-25T10:00:00Z"
    }
  ]
}
```

#### Assign Policy to Agent
```http
POST /policies/{policyId}/agents/{agentId}
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "message": "Policy assigned successfully",
  "policyId": "uuid",
  "policyName": "daily-backup",
  "agentId": "uuid",
  "agentHostname": "backup-server-1"
}
```

#### Remove Policy from Agent
```http
DELETE /policies/{policyId}/agents/{agentId}
X-Tenant-ID: {tenant-uuid}
```

---

### 1.4 Task Distribution

#### Get Pending Tasks for Agent
```http
GET /agents/{agentId}/tasks?limit=10
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "agentId": "uuid",
      "taskType": "backup",
      "policyId": "uuid",
      "status": "pending",
      "repository": "s3:s3.amazonaws.com/bucket",
      "includePaths": {"paths": ["/data"]},
      "excludePaths": {"patterns": ["*.tmp"]},
      "retention": {"keepLast": 7},
      "scheduledAt": "2025-11-25T02:00:00Z",
      "createdAt": "2025-11-25T01:59:00Z"
    }
  ]
}
```

#### Acknowledge Task
```http
POST /agents/{agentId}/tasks/{taskId}/ack
X-Tenant-ID: {tenant-uuid}

{
  "status": "in-progress"
}
```

---

### 1.5 Backup Run History & Logs

#### Get Backup Runs for Agent
```http
GET /agents/{agentId}/backup-runs?status=success&limit=50&offset=0
X-Tenant-ID: {tenant-uuid}
```

**Query Parameters:**
- `status`: success, failed, running
- `limit`: 1-100 (default 50)
- `offset`: Pagination offset

**Response:**
```json
{
  "runs": [
    {
      "id": "uuid",
      "agentId": "uuid",
      "policyId": "uuid",
      "taskId": "uuid",
      "taskType": "backup",
      "status": "success",
      "startTime": "2025-11-25T02:00:00Z",
      "endTime": "2025-11-25T02:15:30Z",
      "duration": 930,
      "bytesProcessed": 1073741824,
      "filesProcessed": 1250,
      "snapshotId": "abc123def456"
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

#### Get Backup Run with Logs
```http
GET /agents/{agentId}/backup-runs/{runId}
X-Tenant-ID: {tenant-uuid}
```

**Response:** Same as above + `logs` field:
```json
{
  "id": "uuid",
  "status": "success",
  "logs": "2025-11-25 02:00:00 Starting backup...\n2025-11-25 02:15:30 Backup complete\n",
  ...
}
```

---

### 1.6 Task Result Submission

#### Submit Task Result
```http
POST /agents/{agentId}/tasks/results
X-Tenant-ID: {tenant-uuid}

{
  "taskId": "uuid",
  "policyId": "uuid",
  "taskType": "backup",
  "status": "success",
  "startTime": "2025-11-25T02:00:00Z",
  "endTime": "2025-11-25T02:15:30Z",
  "duration": 930.5,
  "bytesProcessed": 1073741824,
  "filesProcessed": 1250,
  "snapshotId": "abc123def456",
  "logs": "Backup log output here...",
  "errorMessage": null
}
```

**Note:** Logs can be up to 10MB. Auto-chunked into 1MB pieces.

---

### 1.7 Scheduler Status

#### Get Scheduler Status
```http
GET /scheduler/status
X-Tenant-ID: {tenant-uuid}
```

**Response:**
```json
{
  "running": true,
  "lastRun": "2025-11-25T10:00:00Z",
  "totalRuns": 1000,
  "tasksGenerated": 5000,
  "errorsTotal": 2,
  "lastError": "Invalid cron format for policy xyz",
  "policiesEnabled": 10,
  "upcomingSchedule": [
    {
      "policyId": "uuid",
      "policyName": "daily-backup",
      "taskType": "backup",
      "nextRun": "2025-11-26T02:00:00Z",
      "schedule": "0 2 * * *"
    }
  ],
  "metrics": {
    "tasksGeneratedTotal": 5000,
    "tasksGeneratedByType": {
      "backup": 3000,
      "check": 1500,
      "prune": 500
    },
    "averageProcessingTime": "150ms"
  }
}
```

---

## 2. Data Models Reference

### Agent Status Values
- `online` - Active heartbeat within 5 minutes
- `offline` - No heartbeat for 5+ minutes
- `unknown` - Never received heartbeat

### Task Types
- `backup` - Full or incremental backup
- `check` - Repository integrity check
- `prune` - Clean old snapshots per retention rules

### Task Status Values
- `pending` - Waiting for agent pickup
- `in-progress` - Agent acknowledged and executing
- `completed` - Successfully finished
- `failed` - Execution failed

### Backup Run Status Values
- `running` - Currently executing
- `success` - Completed successfully
- `failed` - Failed with error

### Repository Types
- `s3` - S3-compatible storage
- `rest-server` - Restic REST server
- `filesystem` - Local/NFS filesystem
- `sftp` - SFTP remote server

### Schedule Formats

**Cron (5 fields):**
```
0 2 * * *        # Daily at 2 AM
*/30 * * * *     # Every 30 minutes
0 0 * * 0        # Weekly on Sunday
```

**Interval:**
```
every 6h         # Every 6 hours
every 30m        # Every 30 minutes
every 24h        # Daily
```

---

## 3. Authentication & Multi-Tenancy

### Required Headers
All API requests must include:
```http
X-Tenant-ID: {tenant-uuid}
```

**Note:** Current implementation assumes tenant ID is provided. Future: OAuth2/JWT with tenant extraction.

---

## 4. Error Response Format

All errors follow this structure:
```json
{
  "error": "Human-readable error message",
  "details": "Additional context (optional)"
}
```

**Common HTTP Status Codes:**
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Validation error
- `404 Not Found` - Resource not found
- `409 Conflict` - Duplicate/constraint violation
- `500 Internal Server Error` - Server error

---

## 5. UI Component Recommendations

### 5.1 Agent Overview Page

**Components:**
- Agent list table with status indicators
- Real-time status updates (consider WebSocket for live updates)
- Filter by status (online/offline)
- Search by hostname
- Sort by last heartbeat, status
- Agent detail modal/page showing:
  - Disk usage charts
  - Assigned policies
  - Recent backup runs
  - System info (OS, version, uptime)

**API Calls:**
- `GET /agents` - List with auto-refresh
- `GET /agents/{id}` - Details
- `GET /agents/{id}/policies` - Assigned policies
- `GET /agents/{id}/backup-runs` - Recent runs

---

### 5.2 Policy Management Page

**Components:**
- Policy CRUD forms
- Schedule editor with cron helper
- Repository configuration wizard
- Path editor (include/exclude)
- Retention rules builder
- Policy validation feedback
- Policy preview/test

**API Calls:**
- `GET /policies` - List
- `POST /policies` - Create
- `PUT /policies/{id}` - Update
- `DELETE /policies/{id}` - Delete
- `GET /policies/{id}/agents` - Assigned agents

**Validation:**
Implement client-side validation matching backend rules (see Policy Management section above)

---

### 5.3 Policy-Agent Assignment Page

**Components:**
- Drag-and-drop interface
- Multi-select for bulk assignment
- Visual policy-agent mapping
- Assignment status indicators

**API Calls:**
- `POST /policies/{policyId}/agents/{agentId}` - Assign
- `DELETE /policies/{policyId}/agents/{agentId}` - Remove
- `GET /policies/{policyId}/agents` - List agents
- `GET /agents/{agentId}/policies` - List policies

---

### 5.4 Backup History Page

**Components:**
- Backup run timeline/table
- Filter by agent, policy, status, date range
- Pagination controls
- Log viewer modal with syntax highlighting
- Duration and size charts
- Success rate dashboard

**API Calls:**
- `GET /agents/{id}/backup-runs?status=...&limit=...&offset=...`
- `GET /agents/{id}/backup-runs/{runId}` - With logs

**Features:**
- Infinite scroll or pagination
- Download logs as text file
- Copy logs to clipboard
- Filter by date range picker

---

### 5.5 Scheduler Dashboard

**Components:**
- Scheduler status indicator (running/stopped)
- Upcoming schedule timeline
- Task generation metrics charts
- Error log viewer
- Schedule calendar view

**API Calls:**
- `GET /scheduler/status` - Auto-refresh every 30s

**Visualizations:**
- Timeline showing next 24/48 hours of scheduled tasks
- Pie chart of task types
- Line graph of tasks generated over time
- Error rate trends

---

### 5.6 Dashboard/Home Page

**Components:**
- System health overview
- Agent status summary (X online, Y offline)
- Recent backup activity
- Failed backup alerts
- Scheduler status
- Quick actions (create policy, register agent)

**API Calls:**
- `GET /agents` - Count online/offline
- `GET /agents/{id}/backup-runs?status=failed&limit=10` - Recent failures
- `GET /scheduler/status` - Scheduler health

---

## 6. State Management Recommendations

### Suggested Architecture
- **React + Redux Toolkit** or **Zustand** for state management
- **React Query** or **SWR** for API caching and auto-refresh
- **Axios** for HTTP client with interceptors for tenant header

### Key State Slices
```javascript
{
  agents: {
    list: [],
    selected: null,
    loading: false,
    error: null
  },
  policies: {
    list: [],
    selected: null,
    loading: false,
    error: null
  },
  backupRuns: {
    byAgent: {},
    selected: null,
    loading: false
  },
  scheduler: {
    status: {},
    upcomingSchedule: []
  },
  tenant: {
    id: "uuid",
    name: "My Organization"
  }
}
```

---

## 7. Real-Time Updates (Future Enhancement)

Consider implementing WebSocket for:
- Live agent status changes
- Real-time backup progress
- Scheduler task generation events

**Endpoint (future):** `ws://orchestrator:8080/ws?tenantId={uuid}`

---

## 8. Testing Strategy

### API Mocking
Use MSW (Mock Service Worker) for development:
```javascript
// Mock GET /agents
rest.get('/agents', (req, res, ctx) => {
  return res(
    ctx.json({
      agents: [mockAgent1, mockAgent2]
    })
  );
});
```

### E2E Testing
- Cypress or Playwright for UI flows
- Test complete workflows:
  - Create policy → Assign to agent → View backup run
  - Register agent → View status → Check heartbeat

---

## 9. Design System

### Status Colors
- **Success/Online:** Green (#10B981)
- **Warning:** Yellow (#F59E0B)
- **Error/Offline:** Red (#EF4444)
- **Info:** Blue (#3B82F6)
- **Neutral:** Gray (#6B7280)

### Icons
- Agent: 🖥️ Computer/Server
- Policy: 📋 Clipboard/Document
- Backup: 💾 Save/Upload
- Check: ✓ Checkmark
- Prune: 🗑️ Trash
- Schedule: 📅 Calendar/Clock

---

## 10. Development Checklist

### Phase 1: Core UI (Week 1-2)
- [ ] Setup React + TypeScript project
- [ ] Configure API client with tenant header
- [ ] Implement agent list page
- [ ] Implement policy CRUD pages
- [ ] Create reusable components (tables, forms, modals)

### Phase 2: Advanced Features (Week 3-4)
- [ ] Policy-agent assignment UI
- [ ] Backup history viewer with logs
- [ ] Scheduler dashboard
- [ ] Dashboard/home page

### Phase 3: Polish & Testing (Week 5)
- [ ] Responsive design (mobile/tablet)
- [ ] Dark mode support
- [ ] Accessibility (WCAG 2.1)
- [ ] E2E test coverage
- [ ] Performance optimization

---

## 11. API Client Example

```typescript
// api/client.ts
import axios from 'axios';

const client = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add tenant ID to all requests
client.interceptors.request.use((config) => {
  const tenantId = localStorage.getItem('tenantId');
  if (tenantId) {
    config.headers['X-Tenant-ID'] = tenantId;
  }
  return config;
});

export const agentsAPI = {
  list: () => client.get('/agents'),
  get: (id: string) => client.get(`/agents/${id}`),
  register: (data: AgentRegisterRequest) => 
    client.post('/agents/register', data),
};

export const policiesAPI = {
  list: () => client.get('/policies'),
  create: (data: PolicyCreateRequest) => 
    client.post('/policies', data),
  update: (id: string, data: PolicyUpdateRequest) => 
    client.put(`/policies/${id}`, data),
  delete: (id: string) => 
    client.delete(`/policies/${id}`),
};

// ... more API modules
```

---

## 12. Next Steps

1. **Complete EPIC 13 remaining tasks** (API docs + metrics)
2. **Setup frontend project** (React + TypeScript + Tailwind)
3. **Implement authentication** (if not using X-Tenant-ID header)
4. **Build component library** (following design system)
5. **Integrate with backend APIs**
6. **Add real-time features** (WebSocket for live updates)

---

## Appendix A: Sample Workflows

### Workflow 1: Onboard New Agent
1. Agent installs binary and config
2. Agent starts and calls `POST /agents/register`
3. UI shows new agent in list (status: online)
4. Admin assigns policies via UI (`POST /policies/{id}/agents/{agentId}`)
5. Scheduler generates first task
6. Agent picks up task via polling
7. Agent executes backup and submits result
8. UI shows backup run in history

### Workflow 2: Create Backup Policy
1. Admin opens policy creation form
2. Sets name, schedule, repository, paths, retention
3. UI validates fields client-side
4. Submit `POST /policies`
5. Backend validates and creates policy
6. UI shows new policy in list
7. Admin assigns to agents
8. Scheduler picks up policy on next run

### Workflow 3: View Backup Status
1. User navigates to backup history page
2. UI calls `GET /agents/{id}/backup-runs?limit=50`
3. Display runs in table/timeline
4. User clicks run to view details
5. UI calls `GET /agents/{id}/backup-runs/{runId}`
6. Display logs in modal with scroll/search

---

**Backend Status:** ✅ Ready for UI integration
**API Documentation:** Complete
**Test Coverage:** 558 tests passing
**Next Phase:** Frontend development (EPIC 15-17)
