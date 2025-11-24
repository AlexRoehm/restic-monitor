# EPIC Backlog Overview

# **EPIC 1 — Repository Restructuring & Initial Orchestrator Definition**

### User Stories

1. Define repository structure
2. Update README with architecture direction

---

# **EPIC 2 — High-Level System Architecture Document**

### User Stories

1. Create `/docs/architecture.md` with system overview
2. Define detailed component architecture
3. Produce data flow diagrams and sequence descriptions
4. Document data models & relationships

---

# **EPIC 3 — Database Schema Extension for Agents, Policies, Backup Runs**

### User Stories

1. Define and document required database models
2. Implement GORM models for agents, policies, backup runs
3. Implement database migrations
4. Ensure orchestrator can query new data efficiently
5. Add CI validation for schema drift

---

# **EPIC 4 — API for Agent Registration (Using Existing Authentication)**

### User Stories

1. Define the agent registration API contract
2. Implement router endpoint for agent registration
3. Auto-update metadata on re-registration
4. Return structured registration result
5. Integrate registration with logging and metrics

---

# **EPIC 5 — Heartbeat & Status Reporting**

### User Stories

1. Define heartbeat API contract
2. Implement heartbeat router endpoint
3. Determine online/offline status automatically
4. Store disk and resource information
5. UI-friendly status and metadata calculation
6. Metrics and logging for heartbeats

---

# **EPIC 6 — Policy Management Backend**

### User Stories

1. Define policy data model and contract
2. CRUD API for policies
3. Validate policy fields and schedules (cron)
4. Return policy data for UI consumption
5. Logging & metrics for policy operations

---

# **EPIC 7 — Assign Policies to Agents**

### User Stories

1. Define agent-policy assignment API
2. Implement linking/unlinking of policies to agents
3. Display assigned policies when querying agents
4. Ensure integrity (no duplicate assignments)
5. Logging & metrics for assignment operations

---

# **EPIC 8 — Agent Bootstrap & Installation Mechanism**

### User Stories

1. Define agent configuration format
2. Implement first-run registration logic
3. Implement platform-specific service installation (systemd/launchd/windows service)
4. Store agent ID returned by orchestrator
5. Provide diagnostics/log output for installation

---

# **EPIC 9 — Agent Polling Loop**

### User Stories

1. Define polling interval configuration
2. Implement periodic task check
3. Integrate heartbeat into polling loop
4. Handle network failures gracefully
5. Log and metrics for polling activity

---

# **EPIC 10 — Task Distribution API**

### User Stories

1. Define task schema (backup/check/prune)
2. API endpoint to retrieve pending tasks
3. Orchestrator-side task queue implementation
4. Acknowledge task pickup
5. Logging & metrics for task distribution

---

# **EPIC 11 — Execution of Backup Tasks in Agent**

### User Stories

1. Implement restic backup execution wrapper
2. Implement backup status reporting
3. Handle execution errors and retry logic
4. Validate include/exclude paths locally
5. Produce structured log output

---

# **EPIC 12 — Prune & Check Task Support**

### User Stories

1. Implement restic prune execution
2. Implement restic check execution
3. Handle repository lock conflicts
4. Produce structured results for UI
5. Logging & metrics

---

# **EPIC 13 — Backup Log Ingestion**

### User Stories

1. API endpoint for sending logs
2. Store logs in backup_runs table
3. Implement ingestion of large logs (chunked or streamed)
4. Provide logs via API to UI
5. Log ingestion metrics & retention rules

---

# **EPIC 14 — Policy-Based Scheduler in Orchestrator**

### User Stories

1. Implement cron-based scheduling engine
2. Generate tasks when schedule triggers
3. Prevent duplicate scheduling
4. Expose schedule information via API
5. Logging & metrics for scheduling events

---

# **EPIC 15 — Agent Overview UI**

### User Stories

1. Display list of agents and their status
2. Show agent details (disk, version, last backup)
3. Filter and sort agents
4. Show status transitions and warnings
5. Basic troubleshooting info for each agent

---

# **EPIC 16 — Policy Management UI**

### User Stories

1. Create UI for policy CRUD
2. Cron editor UI
3. Include/exclude path editor
4. Repository backend selector
5. Validation & inline error feedback

---

# **EPIC 17 — Backup History & Logs UI**

### User Stories

1. Display backup runs per agent
2. Show status, duration, statistics
3. Show logs in expandable pane/dialog
4. Filter by agent or policy
5. Display failures with hints

---

# **EPIC 18 — Repository Backend Integration Improvements**

### User Stories

1. Support selection of backend types (rest-server/S3/fs)
2. Centralized repository config management
3. Validate repository connectivity
4. Monitor repository space & latency
5. UI indicators for repo health

---

# **EPIC 19 — Build & Deployment Automation**

### User Stories

1. Build multi-platform agent binaries
2. Build orchestrator Docker images
3. Provide docker-compose.example.yml
4. Provide GitHub Actions for CI/CD
5. Smoke tests on build artifacts

---

# **EPIC 20 — End-to-End Testing & Documentation**

### User Stories

1. Define full system test scenarios
2. Implement E2E invocations of agent + orchestrator
3. Add troubleshooting docs
4. Add full installation guide
5. Prepare final system architecture release notes

---

If you'd like, I can turn this into:

* A **Markdown overview table**
* **GitHub issue templates** for each epic and story
* A **Gantt-style roadmap**
* A **dependency graph** showing epic order

Just tell me what format you prefer!
