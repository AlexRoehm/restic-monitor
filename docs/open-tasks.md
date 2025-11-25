# Open Tasks - EPICs 2-14

**Test Status:** ✅ All 558 tests passing (as of 2025-11-25)

---

## EPIC 2 — High-Level System Architecture Document ✅ COMPLETE

All user stories complete. Documentation exists in `/docs/architecture.md`.

**Status:** No open tasks

---

## EPIC 3 — Database Schema Extension ✅ COMPLETE

All user stories complete:
- ✅ Models defined (Agent, Policy, AgentPolicyLink, BackupRun, Task, BackupRunLog)
- ✅ GORM models implemented with multi-tenancy
- ✅ Migration system implemented (5 migrations)
- ✅ Schema validated with comprehensive tests

**Status:** No open tasks

---

## EPIC 4 — API for Agent Registration ✅ COMPLETE

All user stories complete:
- ✅ Agent registration API contract defined
- ✅ POST /agents/register endpoint implemented
- ✅ Auto-update metadata on re-registration
- ✅ Structured registration response
- ✅ Logging and metrics integrated

**Test Coverage:** 8 tests passing

**Status:** No open tasks

---

## EPIC 5 — Heartbeat & Status Reporting ✅ COMPLETE

All user stories complete:
- ✅ Heartbeat API contract defined
- ✅ POST /agents/{id}/heartbeat endpoint
- ✅ Automatic online/offline status calculation
- ✅ Disk and resource information storage
- ✅ UI-friendly status metadata
- ✅ Comprehensive validation and logging

**Test Coverage:** 9 tests passing

**Status:** No open tasks

---

## EPIC 6 — Policy Management Backend ✅ COMPLETE

All user stories complete:
- ✅ Policy data model with multi-tenancy
- ✅ CRUD API (POST/GET/PUT/DELETE /policies)
- ✅ Comprehensive validation (name, schedule, paths, repository, retention)
- ✅ Schedule validation (cron and interval formats)
- ✅ Repository type validation (S3, rest-server, filesystem, SFTP)
- ✅ UI-friendly error messages

**Test Coverage:** 147 tests passing (validation + CRUD)

**Status:** No open tasks

---

## EPIC 7 — Assign Policies to Agents ✅ COMPLETE

All user stories complete:
- ✅ Agent-policy assignment API defined
- ✅ POST /policies/{policyId}/agents/{agentId} (assign)
- ✅ DELETE /policies/{policyId}/agents/{agentId} (remove)
- ✅ GET /agents/{agentId}/policies (list policies for agent)
- ✅ GET /policies/{policyId}/agents (list agents for policy)
- ✅ Duplicate assignment prevention
- ✅ Cascade delete support
- ✅ Multi-tenant isolation

**Test Coverage:** 16 tests passing

**Status:** No open tasks

---

## EPIC 8 — Agent Bootstrap & Installation ✅ COMPLETE

All user stories complete:
- ✅ YAML configuration format
- ✅ First-run registration logic
- ✅ Platform-specific service installation (systemd, launchd)
- ✅ Agent ID persistence
- ✅ Diagnostics command

**Test Coverage:** Agent tests passing

**Status:** No open tasks

---

## EPIC 9 — Agent Polling Loop ✅ COMPLETE

All user stories complete:
- ✅ Configurable polling interval
- ✅ Periodic task checking
- ✅ Integrated heartbeat
- ✅ Network failure handling with exponential backoff
- ✅ Comprehensive logging and metrics

**Test Coverage:** Agent tests passing

**Status:** No open tasks

---

## EPIC 10 — Task Distribution API ✅ COMPLETE

All user stories complete:
- ✅ Task schema (backup/check/prune) with all fields
- ✅ GET /agents/{id}/tasks endpoint
- ✅ Task queue implementation with status tracking
- ✅ POST /agents/{id}/tasks/{taskId}/ack (acknowledge)
- ✅ Comprehensive logging

**Test Coverage:** 13 tests passing

**Status:** No open tasks

---

## EPIC 11 — Execution of Backup Tasks ✅ COMPLETE

All user stories complete:
- ✅ Restic backup execution wrapper
- ✅ Status reporting to orchestrator
- ✅ Error handling and retry logic
- ✅ Path validation
- ✅ Structured log output

**Test Coverage:** Agent execution tests passing

**Status:** No open tasks

---

## EPIC 12 — Prune & Check Task Support ✅ COMPLETE

All user stories complete:
- ✅ Restic prune execution
- ✅ Restic check execution
- ✅ Repository lock conflict handling
- ✅ Structured results for UI
- ✅ Logging and metrics

**Test Coverage:** Agent tests passing

**Status:** No open tasks

---

## EPIC 13 — Backup Log Ingestion ⚠️ PARTIALLY COMPLETE

**Completed:**
- ✅ 13.2: Log ingestion endpoint (POST /agents/{id}/tasks/results)
- ✅ 13.3: Database persistence (backup_runs table)
- ✅ 13.4: Large log handling (1MB chunking, backup_run_logs table)
- ✅ 13.6: UI retrieval API (GET /agents/{id}/backup-runs, GET /agents/{id}/backup-runs/{runId})

**Test Coverage:** 19 tests passing

**Open Tasks:**
1. **13.1: API Documentation** (Swagger/OpenAPI)
   - Need to generate/update API docs for backup run endpoints
   - Document request/response schemas
   - Add example payloads

2. **13.5: Metrics & Logging**
   - Add Prometheus metrics for log ingestion
   - Track log chunk counts, sizes
   - Monitor retrieval performance

**Estimated Effort:** 4-6 hours

---

## EPIC 14 — Policy-Based Scheduler ✅ COMPLETE

All user stories complete:
- ✅ 14.1: Schedule format specification (cron & interval)
- ✅ 14.2: Schedule parser with validation
- ✅ 14.3: Background scheduler loop
- ✅ 14.4: Missed schedule handling
- ✅ 14.5: Multiple task types (backup/check/prune)
- ✅ 14.6: Scheduler metrics
- ✅ 14.7: UI integration (GET /scheduler/status)

**Test Coverage:** 60 tests passing (31 parsing + 14 metrics + 11 scheduler + 5 API)

**Status:** No open tasks

---

## Summary

### Completion Status by EPIC

| EPIC | Name | Status | Open Tasks |
|------|------|--------|------------|
| 2 | Architecture Document | ✅ Complete | 0 |
| 3 | Database Schema | ✅ Complete | 0 |
| 4 | Agent Registration | ✅ Complete | 0 |
| 5 | Heartbeat & Status | ✅ Complete | 0 |
| 6 | Policy Management | ✅ Complete | 0 |
| 7 | Policy Assignment | ✅ Complete | 0 |
| 8 | Agent Bootstrap | ✅ Complete | 0 |
| 9 | Agent Polling | ✅ Complete | 0 |
| 10 | Task Distribution | ✅ Complete | 0 |
| 11 | Backup Execution | ✅ Complete | 0 |
| 12 | Prune/Check Support | ✅ Complete | 0 |
| 13 | Backup Log Ingestion | ⚠️ Partial | 2 |
| 14 | Policy Scheduler | ✅ Complete | 0 |

### Overall Statistics
- **EPICs Complete:** 12/13
- **EPICs Partial:** 1/13 (92% complete)
- **Total Open Tasks:** 2
- **Test Coverage:** 558 tests passing
- **Code Quality:** All tests green, no failing tests

### Next Priority
Complete EPIC 13 remaining tasks (API docs + metrics) before proceeding to UI development.
