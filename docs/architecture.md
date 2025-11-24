# Restic Monitor Architecture

## Table of Contents

1. [System Overview](#system-overview)
2. [High-Level Architecture](#high-level-architecture)
3. [AE Backend Integration](#ae-backend-integration)
4. [Orchestrator Components](#orchestrator-components)
5. [Agent Components](#agent-components)
6. [Communication Flow](#communication-flow)
7. [Backup Task Lifecycle](#backup-task-lifecycle)
8. [Restic Repository Backends](#restic-repository-backends)
9. [Integration with Existing Monitoring](#integration-with-existing-monitoring)
10. [Database Schema](#database-schema)
11. [API Documentation](#api-documentation)
12. [Security Model](#security-model)
13. [Deployment Architecture](#deployment-architecture)

---

## System Overview

Restic Monitor is a distributed backup management system built on the **AE Backend modular architecture**. It provides centralized monitoring and management of Restic backups across multiple machines through a pull-based agent model.

### Key Characteristics

- **Modular Design**: Follows the AE Backend module pattern with `core.Module` interface
- **Multi-Tenant Ready**: Built on tenant-aware base-server infrastructure
- **Pull-Based Communication**: Agents initiate connections to avoid firewall issues
- **Restic-Native**: Leverages Restic's proven backup technology
- **Event-Driven**: Uses AE Backend's event bus for inter-module communication
- **API-First**: RESTful API with OpenAPI/Swagger documentation

### Current vs Future State

| Component | Current (v0.x) | Future (v1.x+) |
|-----------|----------------|----------------|
| **Architecture** | Standalone orchestrator | AE Backend module |
| **Monitoring** | Direct repository polling | Agent-based reporting |
| **Deployment** | Single Go binary + Vue SPA | Multi-module distributed system |
| **Database** | SQLite | PostgreSQL (via AE Backend) |
| **Authentication** | Environment token | AE Backend JWT + multi-tenant |
| **Task Execution** | Manual Restic operations | Scheduled agent tasks |

---

## High-Level Architecture

### Current Architecture (v0.x - Standalone Orchestrator)

```
┌─────────────────────────────────────┐
│   Restic Monitor (Orchestrator)     │
│                                     │
│  ┌──────────────┐  ┌──────────────┐ │
│  │   Vue UI     │  │   Go API     │ │
│  │  (Vite SPA)  │  │  (Gin HTTP)  │ │
│  │              │  │              │ │
│  │ - Dashboard  │  │ - /status    │ │
│  │ - Backups    │  │ - /snapshots │ │
│  │ - i18n       │  │ - /health    │ │
│  └──────┬───────┘  └──────┬───────┘ │
│         │                 │          │
│         └────────┬────────┘          │
│                  │                   │
│           ┌──────▼──────┐           │
│           │   SQLite    │           │
│           │             │           │
│           │ - targets   │           │
│           │ - snapshots │           │
│           │ - health    │           │
│           └─────────────┘           │
└────────────────┬────────────────────┘
                 │ Direct Restic CLI
                 │ (HTTPS + TLS Cert)
      ┌──────────┼──────────┐
      │          │          │
┌─────▼───┐ ┌───▼────┐ ┌───▼────┐
│ Restic  │ │ Restic │ │ Restic │
│ Repo 1  │ │ Repo 2 │ │ Repo 3 │
│         │ │        │ │        │
│ (S3)    │ │ (SFTP) │ │ (Local)│
└─────────┘ └────────┘ └────────┘
```

### Future Architecture (v1.x+ - AE Backend Module)

```
┌────────────────────────────────────────────────────────────────┐
│                    AE Backend Application                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Bootstrap System                        │  │
│  │  ┌────────┐  ┌──────────┐  ┌────────────┐  ┌──────────┐ │  │
│  │  │ Router │  │ EventBus │  │ ModuleReg. │  │ Database │ │  │
│  │  └────────┘  └──────────┘  └────────────┘  └──────────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                      Core Modules                         │  │
│  │  ┌──────┐  ┌──────────┐  ┌───────┐  ┌──────┐  ┌───────┐ │  │
│  │  │ Base │  │ Customer │  │ Email │  │ PDF  │  │Static │ │  │
│  │  │      │  │          │  │       │  │      │  │       │ │  │
│  │  │Auth  │  │ Plans    │  │ SMTP  │  │ Gen. │  │ JSON  │ │  │
│  │  │Users │  │ Tenants  │  │ Queue │  │ Docs │  │ Files │ │  │
│  │  └──────┘  └──────────┘  └───────┘  └──────┘  └───────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Restic Monitor Module (NEW)                  │  │
│  │                                                            │  │
│  │  ┌────────────────────┐         ┌────────────────────┐   │  │
│  │  │  Orchestrator      │         │   Vue UI           │   │  │
│  │  │  Component         │         │   Component        │   │  │
│  │  │                    │         │                    │   │  │
│  │  │ ┌────────────────┐ │         │ ┌────────────────┐ │   │  │
│  │  │ │ Policy Engine  │ │         │ │ Dashboard      │ │   │  │
│  │  │ │ - Schedules    │ │         │ │ - Backups      │ │   │  │
│  │  │ │ - Retention    │ │         │ │ - Agents       │ │   │  │
│  │  │ │ - Alerts       │ │         │ │ - Logs         │ │   │  │
│  │  │ └────────────────┘ │         │ │ - Policies     │ │   │  │
│  │  │                    │         │ └────────────────┘ │   │  │
│  │  │ ┌────────────────┐ │         │                    │   │  │
│  │  │ │ Task Manager   │ │         │ ┌────────────────┐ │   │  │
│  │  │ │ - Queue        │ │         │ │ i18n (de/en)   │ │   │  │
│  │  │ │ - Distribute   │ │         │ │ - Tailwind v4  │ │   │  │
│  │  │ │ - Monitor      │ │         │ │ - DaisyUI v5   │ │   │  │
│  │  │ └────────────────┘ │         │ └────────────────┘ │   │  │
│  │  │                    │         └────────────────────┘   │  │
│  │  │ ┌────────────────┐ │                                  │  │
│  │  │ │ Agent Manager  │ │         ┌────────────────────┐   │  │
│  │  │ │ - Registry     │ │         │ API Routes         │   │  │
│  │  │ │ - Health       │ │         │ /api/v1/restic/... │   │  │
│  │  │ │ - Auth Tokens  │ │         │                    │   │  │
│  │  │ └────────────────┘ │         │ - /agents          │   │  │
│  │  │                    │         │ - /policies        │   │  │
│  │  │ ┌────────────────┐ │         │ - /tasks           │   │  │
│  │  │ │ Monitor Service│ │         │ - /snapshots       │   │  │
│  │  │ │ (Current)      │ │         │ - /repositories    │   │  │
│  │  │ └────────────────┘ │         └────────────────────┘   │  │
│  │  └────────────────────┘                                  │  │
│  │                                                            │  │
│  │  Entities: Target, Snapshot, Agent, Task, Policy          │  │
│  │  Events: BackupCompleted, HealthChanged, AgentRegistered  │  │
│  │  Services: MonitorService, TaskService, PolicyService     │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────┬──────────────────────────────────────┘
                          │ PostgreSQL
                          │ (Multi-tenant)
                          │
                ┌─────────▼─────────┐
                │   Database        │
                │                   │
                │ - users           │
                │ - tenants         │
                │ - restic_targets  │
                │ - restic_agents   │
                │ - restic_tasks    │
                │ - restic_snapshots│
                └───────────────────┘

                          │ HTTPS Pull-Based
                          │ (JWT Auth)
                          │
            ┌─────────────┼─────────────┐
            │             │             │
        ┌───▼────┐    ┌──▼─────┐   ┌───▼────┐
        │ Agent  │    │ Agent  │   │ Agent  │
        │        │    │        │   │        │
        │ ┌────┐ │    │ ┌────┐ │   │ ┌────┐ │
        │ │Res │ │    │ │Res │ │   │ │Res │ │
        │ │tic │ │    │ │tic │ │   │ │tic │ │
        │ └─┬──┘ │    │ └─┬──┘ │   │ └─┬──┘ │
        └───┼────┘    └───┼────┘   └───┼────┘
            │             │            │
        ┌───▼────┐    ┌──▼─────┐  ┌───▼────┐
        │ Backup │    │ Backup │  │ Backup │
        │ Repo   │    │ Repo   │  │ Repo   │
        │ (S3)   │    │ (SFTP) │  │(Local) │
        └────────┘    └────────┘  └────────┘
```

---

## AE Backend Integration

### Module Implementation

Restic Monitor integrates as a standard **AE Backend module**, implementing the `core.Module` interface:

```go
package restic

import (
    "context"
    "github.com/ae-base-server/pkg/core"
    "gorm.io/gorm"
)

// ResticModule provides backup monitoring and management
type ResticModule struct {
    db               *gorm.DB
    monitorService   *services.MonitorService
    taskService      *services.TaskService
    agentManager     *services.AgentManager
    policyEngine     *services.PolicyEngine
    routeProvider    *routes.RouteProvider
    eventHandlers    []core.EventHandler
}

func NewResticModule() core.Module {
    return &ResticModule{}
}

func (m *ResticModule) Name() string {
    return "restic"
}

func (m *ResticModule) Version() string {
    return "1.0.0"
}

func (m *ResticModule) Dependencies() []string {
    return []string{"base"} // Depends on base module for auth/users
}

func (m *ResticModule) Initialize(ctx core.ModuleContext) error {
    ctx.Logger.Info("Initializing Restic Monitor module...")
    
    m.db = ctx.DB
    
    // Initialize services
    m.monitorService = services.NewMonitorService(ctx.DB)
    m.taskService = services.NewTaskService(ctx.DB, ctx.EventBus)
    m.agentManager = services.NewAgentManager(ctx.DB)
    m.policyEngine = services.NewPolicyEngine(ctx.DB)
    
    // Initialize handlers
    agentHandler := handlers.NewAgentHandler(m.agentManager)
    policyHandler := handlers.NewPolicyHandler(m.policyEngine)
    taskHandler := handlers.NewTaskHandler(m.taskService)
    monitorHandler := handlers.NewMonitorHandler(m.monitorService)
    
    // Initialize route provider
    m.routeProvider = routes.NewRouteProvider(
        agentHandler,
        policyHandler,
        taskHandler,
        monitorHandler,
        ctx.DB,
    )
    
    // Initialize event handlers
    m.eventHandlers = []core.EventHandler{
        events.NewBackupCompletedHandler(ctx.Logger),
        events.NewHealthChangedHandler(ctx.Logger, ctx.EventBus),
        events.NewAgentRegisteredHandler(ctx.Logger),
    }
    
    ctx.Logger.Info("Restic Monitor module initialized successfully")
    return nil
}

func (m *ResticModule) Start(ctx context.Context) error {
    // Start background monitoring (for current direct-poll targets)
    return m.monitorService.StartMonitoring(ctx)
}

func (m *ResticModule) Stop(ctx context.Context) error {
    // Stop background monitoring
    return m.monitorService.StopMonitoring(ctx)
}

func (m *ResticModule) Entities() []core.Entity {
    return []core.Entity{
        entities.NewTargetEntity(),
        entities.NewAgentEntity(),
        entities.NewSnapshotEntity(),
        entities.NewTaskEntity(),
        entities.NewPolicyEntity(),
    }
}

func (m *ResticModule) Routes() []core.RouteProvider {
    return []core.RouteProvider{m.routeProvider}
}

func (m *ResticModule) EventHandlers() []core.EventHandler {
    return m.eventHandlers
}
```

### Server Integration

In the AE Backend server's `main.go`:

```go
package main

import (
    "context"
    "log"
    
    "github.com/ae-base-server/modules/base"
    "github.com/ae-base-server/modules/customer"
    "github.com/ae-base-server/modules/email"
    "github.com/ae-base-server/pkg/bootstrap"
    "github.com/ae-base-server/pkg/config"
    "github.com/ae-base-server/pkg/core"
    restic "github.com/ae-backend/restic-monitor"
    "github.com/joho/godotenv"
)

// @tag.name restic
// @tag.description [Restic Module] Backup monitoring and management

func main() {
    godotenv.Load()
    
    cfg := config.Load()
    app := bootstrap.NewApplication(cfg)
    
    // Register modules in dependency order
    modules := []core.Module{
        base.NewBaseModule(),         // Auth, users, tenants
        customer.NewCustomerModule(), // Customer management
        email.NewEmailModule(),       // Email notifications
        restic.NewResticModule(),     // Backup monitoring (NEW)
    }
    
    for _, module := range modules {
        if err := app.RegisterModule(module); err != nil {
            log.Fatalf("Failed to register module: %v", err)
        }
    }
    
    if err := app.Initialize(); err != nil {
        log.Fatal("Failed to initialize:", err)
    }
    
    if err := app.Start(context.Background()); err != nil {
        log.Fatal("Failed to start:", err)
    }
}
```

### Benefits of AE Backend Integration

1. **Multi-Tenancy**: Automatic tenant isolation for backups
2. **Authentication**: JWT-based auth with user roles
3. **Event System**: Pub/sub for backup notifications
4. **Service Registry**: Dependency injection
5. **Middleware**: CORS, logging, rate limiting
6. **Database Migrations**: Automatic schema management
7. **Swagger Integration**: Auto-generated API docs
8. **Health Checks**: System monitoring endpoints

---

## Orchestrator Components

The orchestrator is the central management system running as an AE Backend module.

### Component Architecture

```
Orchestrator Module
├── entities/
│   ├── target.go          # Backup target configuration
│   ├── agent.go           # Agent registration & metadata
│   ├── snapshot.go        # Snapshot records
│   ├── task.go            # Backup/check/prune tasks
│   └── policy.go          # Backup policies & schedules
├── services/
│   ├── monitor_service.go # Current polling monitor
│   ├── agent_manager.go   # Agent lifecycle management
│   ├── task_service.go    # Task creation & distribution
│   ├── policy_engine.go   # Schedule & retention policies
│   └── notification.go    # Alerts via email module
├── handlers/
│   ├── agent_handler.go   # Agent registration & heartbeat
│   ├── policy_handler.go  # Policy CRUD
│   ├── task_handler.go    # Task status & logs
│   └── monitor_handler.go # Current monitoring endpoints
├── routes/
│   └── routes.go          # Route registration
├── events/
│   ├── backup_completed.go
│   ├── health_changed.go
│   └── agent_registered.go
└── module.go              # Module definition
```

### Core Services

#### 1. Monitor Service (Current)

**Purpose**: Direct polling of Restic repositories (existing functionality)

```go
type MonitorService struct {
    db       *gorm.DB
    targets  []config.Target
    interval time.Duration
    stopChan chan struct{}
}

// StartMonitoring polls repositories on interval
func (s *MonitorService) StartMonitoring(ctx context.Context) error

// CheckTarget polls a single Restic repository
func (s *MonitorService) CheckTarget(target config.Target) error

// GetSnapshots retrieves snapshots from repository
func (s *MonitorService) GetSnapshots(target config.Target) ([]Snapshot, error)

// CheckHealth verifies repository health
func (s *MonitorService) CheckHealth(target config.Target) (bool, error)
```

#### 2. Agent Manager (Future)

**Purpose**: Manage agent lifecycle and authentication

```go
type AgentManager struct {
    db       *gorm.DB
    tokenSvc *auth.TokenService
}

// RegisterAgent creates new agent with token
func (m *AgentManager) RegisterAgent(req AgentRegistration) (*Agent, error)

// AuthenticateAgent validates agent token
func (m *AgentManager) AuthenticateAgent(token string) (*Agent, error)

// UpdateHeartbeat records agent health
func (m *AgentManager) UpdateHeartbeat(agentID string, status AgentStatus) error

// GetAgentStatus retrieves current agent state
func (m *AgentManager) GetAgentStatus(agentID string) (*AgentStatus, error)

// RevokeAgent disables agent and invalidates token
func (m *AgentManager) RevokeAgent(agentID string) error
```

#### 3. Task Service (Future)

**Purpose**: Create, queue, and distribute backup tasks to agents

```go
type TaskService struct {
    db       *gorm.DB
    eventBus core.EventBus
}

// CreateTask generates a new backup/check/prune task
func (s *TaskService) CreateTask(req TaskRequest) (*Task, error)

// GetPendingTasks retrieves tasks for an agent (long-polling)
func (s *TaskService) GetPendingTasks(agentID string, timeout time.Duration) ([]Task, error)

// UpdateTaskStatus records task progress
func (s *TaskService) UpdateTaskStatus(taskID string, status TaskStatus) error

// StreamLogs receives real-time logs from agent
func (s *TaskService) StreamLogs(taskID string, logs []LogEntry) error

// CancelTask stops a running task
func (s *TaskService) CancelTask(taskID string) error
```

#### 4. Policy Engine (Future)

**Purpose**: Define and enforce backup policies

```go
type PolicyEngine struct {
    db *gorm.DB
}

// CreatePolicy defines a new backup policy
func (e *PolicyEngine) CreatePolicy(policy Policy) error

// EvaluatePolicy checks if backups comply with policy
func (e *PolicyEngine) EvaluatePolicy(targetID string) (*PolicyResult, error)

// GetSchedule calculates next backup time
func (e *PolicyEngine) GetSchedule(policyID string) (time.Time, error)

// ApplyRetention prunes old snapshots per policy
func (e *PolicyEngine) ApplyRetention(targetID string) error
```

### API Routes

All routes are prefixed with `/api/v1/restic/`:

**Current (v0.x - Monitoring):**
- `GET /status` - List all backup targets
- `GET /status/:name` - Get specific target status
- `GET /status/:name/snapshots` - List snapshots
- `GET /status/:name/snapshots/:id/files` - Get file list
- `GET /health` - Orchestrator health check

**Future (v1.x+ - Full Management):**

**Agent Management:**
- `POST /agents/register` - Register new agent
- `POST /agents/:id/heartbeat` - Agent health ping
- `GET /agents` - List all agents
- `GET /agents/:id` - Get agent details
- `DELETE /agents/:id` - Revoke agent

**Task Management:**
- `GET /agents/:id/tasks` - Poll for pending tasks (long-polling)
- `POST /tasks/:id/status` - Update task status
- `POST /tasks/:id/logs` - Stream task logs
- `POST /tasks` - Manually create task
- `DELETE /tasks/:id` - Cancel task

**Policy Management:**
- `GET /policies` - List backup policies
- `POST /policies` - Create policy
- `PUT /policies/:id` - Update policy
- `DELETE /policies/:id` - Delete policy
- `GET /policies/:id/compliance` - Check policy compliance

**Repository Management:**
- `GET /repositories` - List all repositories
- `POST /repositories` - Add repository
- `PUT /repositories/:id` - Update repository
- `GET /repositories/:id/snapshots` - List snapshots

---

## Agent Components

The agent is a lightweight daemon running on machines to be backed up.

### Agent Architecture

```
Agent Binary
├── cmd/
│   └── agent/
│       └── main.go           # Entry point
├── pkg/
│   ├── client/
│   │   └── orchestrator.go   # API client
│   ├── executor/
│   │   ├── restic.go         # Restic command wrapper
│   │   ├── backup.go         # Backup execution
│   │   ├── check.go          # Repository checks
│   │   └── prune.go          # Cleanup operations
│   ├── scheduler/
│   │   └── poller.go         # Task polling
│   ├── heartbeat/
│   │   └── monitor.go        # Health monitoring
│   └── config/
│       └── config.go         # Agent configuration
├── plugins/
│   ├── mysql/
│   │   └── dump.go           # MySQL backup plugin
│   ├── postgres/
│   │   └── dump.go           # PostgreSQL backup plugin
│   └── docker/
│       └── volume.go         # Docker volume plugin
└── README.md
```

### Core Components

#### 1. Orchestrator Client

**Purpose**: Communicate with central orchestrator

```go
type OrchestratorClient struct {
    baseURL string
    token   string
    httpClient *http.Client
}

// Register registers agent with orchestrator
func (c *OrchestratorClient) Register(info AgentInfo) (*AgentCredentials, error)

// PollTasks long-polls for pending tasks
func (c *OrchestratorClient) PollTasks(timeout time.Duration) ([]Task, error)

// UpdateTaskStatus sends task progress updates
func (c *OrchestratorClient) UpdateTaskStatus(taskID string, status TaskStatus) error

// StreamLogs sends real-time logs
func (c *OrchestratorClient) StreamLogs(taskID string, logs chan LogEntry) error

// Heartbeat sends periodic health status
func (c *OrchestratorClient) Heartbeat(status HealthStatus) error
```

#### 2. Task Executor

**Purpose**: Execute Restic operations

```go
type ResticExecutor struct {
    binary     string
    cacheDir   string
    configDir  string
}

// Backup creates a new snapshot
func (e *ResticExecutor) Backup(ctx context.Context, opts BackupOptions) error

// Check verifies repository integrity
func (e *ResticExecutor) Check(ctx context.Context, repo Repository) error

// Prune removes old snapshots
func (e *ResticExecutor) Prune(ctx context.Context, policy RetentionPolicy) error

// Forget deletes specific snapshots
func (e *ResticExecutor) Forget(ctx context.Context, snapshotIDs []string) error

// Snapshots lists all snapshots
func (e *ResticExecutor) Snapshots(ctx context.Context) ([]Snapshot, error)
```

#### 3. Scheduler

**Purpose**: Poll orchestrator for tasks

```go
type Scheduler struct {
    client       *OrchestratorClient
    executor     *ResticExecutor
    pollInterval time.Duration
    maxConcurrent int
}

// Start begins task polling loop
func (s *Scheduler) Start(ctx context.Context) error

// processTask executes a single task
func (s *Scheduler) processTask(ctx context.Context, task Task) error
```

#### 4. Heartbeat Monitor

**Purpose**: Track and report agent health

```go
type HealthMonitor struct {
    client   *OrchestratorClient
    interval time.Duration
}

// Start begins health monitoring
func (m *HealthMonitor) Start(ctx context.Context) error

// collectMetrics gathers system statistics
func (m *HealthMonitor) collectMetrics() HealthMetrics
```

### Agent Configuration

```yaml
# /etc/restic-agent/config.yaml

orchestrator:
  url: https://restic-monitor.example.com
  token: <agent-registration-token>
  
agent:
  id: server-01
  hostname: web-server
  poll_interval: 60s
  heartbeat_interval: 30s
  max_concurrent_tasks: 2

restic:
  binary: /usr/bin/restic
  cache_dir: /var/cache/restic
  config_dir: /etc/restic

backup:
  sources:
    - path: /home
      policy: daily
    - path: /etc
      policy: hourly
  
  exclude:
    - /home/*/.cache
    - /home/*/tmp
    - "*.log"

repository:
  url: s3:s3.amazonaws.com/my-backup-bucket
  password_file: /etc/restic/password
  
plugins:
  mysql:
    enabled: true
    host: localhost
    user: backup
    password_file: /etc/mysql/backup.pass
    databases:
      - myapp_production
  
  postgres:
    enabled: false
```

---

## Communication Flow

### Pull-Based Architecture

Agents initiate all connections to avoid firewall issues:

```
Agent                    Orchestrator
  │                           │
  │─────── Register ─────────>│
  │<──── Token + AgentID ─────│
  │                           │
  │                           │
  ├─── Heartbeat Loop ────────┤
  │                           │
  │──── Poll Tasks (60s) ────>│
  │<──── [Empty] ─────────────│
  │                           │
  │──── Poll Tasks ──────────>│
  │<──── Task: Backup ────────│
  │                           │
  ├─── Execute Backup ────────┤
  │                           │
  │──── Status: Running ─────>│
  │──── Stream Logs ─────────>│
  │──── Status: Success ─────>│
  │                           │
  │──── Poll Tasks ──────────>│
  │<──── Task: Check ─────────│
  │                           │
```

### Sequence Diagrams

#### Agent Registration

```
Agent                Orchestrator          Database
  │                       │                   │
  │ POST /agents/register │                   │
  ├──────────────────────>│                   │
  │ {hostname, version}   │                   │
  │                       │ Create Agent      │
  │                       ├──────────────────>│
  │                       │ Generate Token    │
  │                       │<──────────────────│
  │                       │                   │
  │ {id, token}           │                   │
  │<──────────────────────┤                   │
  │                       │                   │
```

#### Task Execution

```
Agent            Orchestrator         Database        Restic
  │                   │                  │              │
  │ GET /tasks        │                  │              │
  ├──────────────────>│ Query Pending    │              │
  │ (long-poll 30s)   ├─────────────────>│              │
  │                   │ Task Found       │              │
  │                   │<─────────────────┤              │
  │ {task: backup}    │                  │              │
  │<──────────────────┤                  │              │
  │                   │                  │              │
  │ POST /tasks/:id/status               │              │
  ├──────────────────>│ Update Status    │              │
  │ {status: running} ├─────────────────>│              │
  │                   │                  │              │
  │ Execute Backup    │                  │              │
  ├──────────────────────────────────────┼─────────────>│
  │                   │                  │ Create       │
  │                   │                  │ Snapshot     │
  │                   │                  │<─────────────┤
  │                   │                  │              │
  │ POST /tasks/:id/logs                 │              │
  ├──────────────────>│ Store Logs       │              │
  │ {logs: [...]}     ├─────────────────>│              │
  │                   │                  │              │
  │ POST /tasks/:id/status               │              │
  ├──────────────────>│ Update Status    │              │
  │ {status: success} ├─────────────────>│              │
  │                   │ Emit Event       │              │
  │                   │ BackupCompleted  │              │
  │                   │                  │              │
```

#### Long-Polling Mechanism

```
Agent                    Orchestrator
  │                           │
  │ GET /agents/:id/tasks     │
  ├──────────────────────────>│
  │ (Connection held open)    │ Query DB for pending tasks
  │                           ├─┐
  │                           │ │ No tasks found
  │                           │<┘
  │                           │ Wait (timeout: 30s)
  │                           │
  │                           │ New task arrives
  │                           │ (via policy engine)
  │                           │
  │ [{task}]                  │
  │<──────────────────────────┤
  │                           │
```

---

## Backup Task Lifecycle

### Task States

```
Created → Queued → Assigned → Running → Success
                                      → Failed
                                      → Cancelled
```

### State Transitions

```
┌─────────┐
│ Created │  Policy engine creates task
└────┬────┘
     │
     v
┌─────────┐
│  Queued │  Waiting for agent to poll
└────┬────┘
     │
     v
┌──────────┐
│ Assigned │  Agent retrieved task
└────┬─────┘
     │
     v
┌─────────┐
│ Running │  Agent executing task
└────┬────┘
     │
     ├────────> Success (Snapshot created)
     │
     ├────────> Failed (Error occurred)
     │
     └────────> Cancelled (User/system stopped)
```

### Task Types

#### 1. Backup Task

**Trigger**: Policy schedule or manual request

```json
{
  "id": "task-123",
  "type": "backup",
  "agent_id": "agent-456",
  "created_at": "2025-11-23T10:00:00Z",
  "parameters": {
    "sources": ["/home", "/etc"],
    "exclude": ["*.log", ".cache"],
    "tags": ["auto", "daily"],
    "retention": {
      "keep_daily": 7,
      "keep_weekly": 4,
      "keep_monthly": 12
    }
  }
}
```

**Agent Actions**:
1. Validate source paths exist
2. Run pre-backup plugins (database dumps)
3. Execute `restic backup`
4. Stream logs to orchestrator
5. Report snapshot ID on success
6. Run post-backup cleanup

#### 2. Check Task

**Trigger**: Policy schedule or health alert

```json
{
  "id": "task-789",
  "type": "check",
  "agent_id": "agent-456",
  "created_at": "2025-11-23T11:00:00Z",
  "parameters": {
    "read_data": true,
    "check_unused": true
  }
}
```

**Agent Actions**:
1. Execute `restic check --read-data`
2. Report integrity status
3. Alert on errors

#### 3. Prune Task

**Trigger**: Policy retention rules

```json
{
  "id": "task-abc",
  "type": "prune",
  "agent_id": "agent-456",
  "created_at": "2025-11-23T12:00:00Z",
  "parameters": {
    "retention": {
      "keep_daily": 7,
      "keep_weekly": 4,
      "keep_monthly": 12,
      "keep_yearly": 3
    }
  }
}
```

**Agent Actions**:
1. Execute `restic forget` with retention policy
2. Execute `restic prune` to remove data
3. Report space reclaimed

---

## Restic Repository Backends

Restic supports multiple storage backends. The orchestrator and agents must support all common backends.

### Supported Backends

#### 1. Local Filesystem

```yaml
repository:
  type: local
  path: /mnt/backup
  
# Restic command
restic -r /mnt/backup backup /home
```

**Use Case**: Direct-attached storage, NAS mounts

#### 2. SFTP

```yaml
repository:
  type: sftp
  host: backup-server.example.com
  user: restic
  path: /backups/server-01
  key_file: /etc/restic/ssh_key
  
# Restic command
restic -r sftp:restic@backup-server.example.com:/backups/server-01 backup /home
```

**Use Case**: Remote servers with SSH access

#### 3. S3-Compatible (AWS S3, MinIO, Wasabi, etc.)

```yaml
repository:
  type: s3
  endpoint: s3.amazonaws.com
  bucket: my-backup-bucket
  region: us-east-1
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key_file: /etc/restic/s3-secret
  
# Restic command
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=<from file>
restic -r s3:s3.amazonaws.com/my-backup-bucket backup /home
```

**Use Case**: Cloud storage, object storage

#### 4. REST Server

```yaml
repository:
  type: rest
  url: https://restic-rest.example.com/
  username: restic
  password_file: /etc/restic/rest-password
  
# Restic command
restic -r rest:https://restic@restic-rest.example.com/ backup /home
```

**Use Case**: Self-hosted Restic REST server

#### 5. Azure Blob Storage

```yaml
repository:
  type: azure
  account_name: myaccount
  container: backups
  account_key_file: /etc/restic/azure-key
  
# Restic command
export AZURE_ACCOUNT_NAME=myaccount
export AZURE_ACCOUNT_KEY=<from file>
restic -r azure:backups:/ backup /home
```

**Use Case**: Microsoft Azure cloud

#### 6. Google Cloud Storage

```yaml
repository:
  type: gcs
  bucket: my-backup-bucket
  project_id: my-project
  credentials_file: /etc/restic/gcs-credentials.json
  
# Restic command
export GOOGLE_PROJECT_ID=my-project
export GOOGLE_APPLICATION_CREDENTIALS=/etc/restic/gcs-credentials.json
restic -r gs:my-backup-bucket:/ backup /home
```

**Use Case**: Google Cloud Platform

### Repository Configuration in Orchestrator

The orchestrator stores repository configuration per target/agent:

```go
type Repository struct {
    ID                string
    AgentID           string
    Type              string // "local", "sftp", "s3", "rest", "azure", "gcs"
    URL               string // Full Restic URL
    PasswordFile      string // Path to repository password
    EnvironmentVars   map[string]string // Backend-specific env vars
    TLSCertFile       string // Optional TLS certificate
}
```

Agents receive repository configuration via task parameters and set up the environment before executing Restic commands.

---

## Integration with Existing Monitoring

The current monitoring system (direct repository polling) will be **preserved and enhanced** during the transition to agent-based architecture.

### Hybrid Mode (Transition Period)

```
┌────────────────────────────────────┐
│  Orchestrator Module               │
│                                    │
│  ┌──────────────────────────────┐ │
│  │  Monitor Service             │ │
│  │  (Current Direct Polling)    │ │
│  │                              │ │
│  │  - Poll repositories         │ │
│  │  - Check health              │ │
│  │  - List snapshots            │ │
│  └──────┬───────────────────────┘ │
│         │                          │
│         │ Coexists with            │
│         │                          │
│  ┌──────▼───────────────────────┐ │
│  │  Agent Manager               │ │
│  │  (New Agent-Based)           │ │
│  │                              │ │
│  │  - Register agents           │ │
│  │  - Distribute tasks          │ │
│  │  - Receive reports           │ │
│  └──────────────────────────────┘ │
└────────────────────────────────────┘
          │              │
          │              │
     Direct Poll    Agent Reports
          │              │
          v              v
    ┌─────────┐    ┌─────────┐
    │ Legacy  │    │ Agent-  │
    │ Repos   │    │ Managed │
    └─────────┘    └─────────┘
```

### Migration Path

**Phase 1 - Current State** (Completed)
- ✅ Direct repository polling
- ✅ SQLite database
- ✅ Vue frontend
- ✅ REST API with Swagger

**Phase 2 - AE Backend Integration** (Q1 2026)
- [ ] Implement `core.Module` interface
- [ ] Migrate to PostgreSQL via AE Backend
- [ ] Add multi-tenant support
- [ ] Integrate with AE Backend auth
- [ ] Event-driven health checks

**Phase 3 - Agent Development** (Q2 2026)
- [ ] Agent binary development
- [ ] Agent registration API
- [ ] Task distribution system
- [ ] Hybrid mode (direct + agent)

**Phase 4 - Full Agent System** (Q3-Q4 2026)
- [ ] Plugin system
- [ ] Advanced scheduling
- [ ] Policy engine
- [ ] Deprecate direct polling

### Data Flow - Hybrid Mode

```
┌──────────────────────────────────────────────────────┐
│                   Database (PostgreSQL)               │
│                                                       │
│  restic_targets                                       │
│  ├── id                                               │
│  ├── tenant_id                                        │
│  ├── name                                             │
│  ├── mode (direct | agent)  ← Determines source      │
│  ├── repository_url                                   │
│  └── agent_id (nullable)                              │
│                                                       │
│  restic_snapshots                                     │
│  ├── id                                               │
│  ├── target_id                                        │
│  ├── snapshot_id                                      │
│  ├── source (direct_poll | agent_report)             │
│  └── created_at                                       │
└──────────────────────────────────────────────────────┘
                     ▲              ▲
                     │              │
            Direct Poll         Agent Report
                     │              │
         ┌───────────┴──┐      ┌────┴──────────┐
         │ MonitorSvc   │      │ TaskService   │
         │ (Legacy)     │      │ (New)         │
         └──────────────┘      └───────────────┘
```

### API Compatibility

**Existing endpoints remain unchanged:**
- `/api/v1/restic/status` - Works with both modes
- `/api/v1/restic/status/:name` - Auto-detects source
- `/api/v1/restic/status/:name/snapshots` - Unified view

**Target response includes source:**
```json
{
  "name": "server-01",
  "mode": "agent",
  "agent_id": "agent-456",
  "status": "healthy",
  "last_snapshot": "2025-11-23T10:00:00Z",
  "snapshot_age_hours": 2,
  "source": "agent_report"
}
```

---

## Database Schema

This section defines the complete database schema for the restic-monitor system, including both current (v0.x) and future (v1.x+) models.

### Entity Relationship Diagram

```mermaid
erDiagram
    TENANTS ||--o{ USERS : has
    TENANTS ||--o{ AGENTS : has
    TENANTS ||--o{ POLICIES : has
    TENANTS ||--o{ TARGETS : has
    TENANTS ||--o{ REPOSITORIES : has
    
    AGENTS ||--o{ AGENT_POLICY_LINKS : has
    POLICIES ||--o{ AGENT_POLICY_LINKS : has
    
    AGENTS ||--o{ BACKUP_RUNS : executes
    POLICIES ||--o{ BACKUP_RUNS : defines
    
    TARGETS }o--|| REPOSITORIES : "stores in"
    TARGETS }o--o| AGENTS : "managed by"
    TARGETS }o--o| POLICIES : "follows"
    
    TARGETS ||--o{ SNAPSHOTS : has
    TARGETS ||--o{ HEALTH_CHECKS : has
    
    BACKUP_RUNS ||--o{ SNAPSHOTS : creates
    BACKUP_RUNS ||--o{ BACKUP_RUN_LOGS : has
    
    TENANTS {
        uuid id PK
        string name
        timestamp created_at
        timestamp updated_at
    }
    
    USERS {
        uuid id PK
        uuid tenant_id FK
        string email
        string password_hash
        string role
        timestamp created_at
        timestamp updated_at
    }
    
    AGENTS {
        uuid id PK
        uuid tenant_id FK
        string hostname
        string os
        string arch
        string version
        string status
        timestamp last_seen_at
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }
    
    POLICIES {
        uuid id PK
        uuid tenant_id FK
        string name
        string schedule
        jsonb include_paths
        jsonb exclude_paths
        string repository_url
        string repository_type
        jsonb repository_config
        jsonb retention_rules
        boolean enabled
        timestamp created_at
        timestamp updated_at
    }
    
    AGENT_POLICY_LINKS {
        uuid agent_id FK
        uuid policy_id FK
        timestamp created_at
    }
    
    TARGETS {
        uuid id PK
        uuid tenant_id FK
        string name
        string mode
        uuid repository_id FK
        uuid agent_id FK
        uuid policy_id FK
        boolean disabled
        timestamp created_at
        timestamp updated_at
    }
    
    REPOSITORIES {
        uuid id PK
        uuid tenant_id FK
        string type
        string url
        string password_file
        string cert_file
        jsonb environment_vars
        timestamp created_at
        timestamp updated_at
    }
    
    SNAPSHOTS {
        uuid id PK
        uuid tenant_id FK
        uuid target_id FK
        uuid backup_run_id FK
        string snapshot_id
        string short_id
        timestamp time
        string hostname
        string username
        jsonb paths
        jsonb tags
        string parent
        string tree
        string source
        int file_count
        timestamp created_at
    }
    
    BACKUP_RUNS {
        uuid id PK
        uuid tenant_id FK
        uuid agent_id FK
        uuid policy_id FK
        uuid target_id FK
        timestamp start_time
        timestamp end_time
        string status
        string error_message
        int files_new
        int files_changed
        int files_unmodified
        int dirs_new
        int dirs_changed
        int dirs_unmodified
        bigint data_added
        bigint total_files_processed
        bigint total_bytes_processed
        float duration_seconds
        string snapshot_id
        timestamp created_at
        timestamp updated_at
    }
    
    BACKUP_RUN_LOGS {
        uuid id PK
        uuid backup_run_id FK
        timestamp timestamp
        string level
        text message
        timestamp created_at
    }
    
    HEALTH_CHECKS {
        uuid id PK
        uuid tenant_id FK
        uuid target_id FK
        boolean healthy
        text error_message
        int snapshot_age_hours
        timestamp checked_at
    }
```

### Current Schema (SQLite - v0.x)

```sql
CREATE TABLE targets (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    repository_url TEXT NOT NULL,
    password_file TEXT,
    cert_file TEXT,
    disabled BOOLEAN DEFAULT 0
);

CREATE TABLE snapshots (
    id INTEGER PRIMARY KEY,
    target_id INTEGER,
    snapshot_id TEXT NOT NULL,
    short_id TEXT,
    time TIMESTAMP,
    hostname TEXT,
    username TEXT,
    paths TEXT, -- JSON array
    tags TEXT,  -- JSON array
    parent TEXT,
    tree TEXT,
    FOREIGN KEY (target_id) REFERENCES targets(id)
);

CREATE TABLE health_checks (
    id INTEGER PRIMARY KEY,
    target_id INTEGER,
    healthy BOOLEAN,
    error_message TEXT,
    checked_at TIMESTAMP,
    FOREIGN KEY (target_id) REFERENCES targets(id)
);
```

### Future Schema (PostgreSQL via AE Backend - v1.x+)

#### Core Model Descriptions

##### 1. Agent Model

**Purpose**: Represents a backup agent running on a target machine.

**Table**: `agents`

**Fields**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique agent identifier |
| `tenant_id` | UUID | FOREIGN KEY, NOT NULL | Multi-tenant isolation |
| `hostname` | VARCHAR(255) | NOT NULL | Machine hostname |
| `os` | VARCHAR(50) | NOT NULL | Operating system (linux, windows, darwin) |
| `arch` | VARCHAR(50) | NOT NULL | CPU architecture (amd64, arm64) |
| `version` | VARCHAR(50) | NOT NULL | Agent version (semver) |
| `status` | VARCHAR(50) | NOT NULL | Current status (online, offline, error) |
| `last_seen_at` | TIMESTAMP | NULL | Last heartbeat timestamp |
| `metadata` | JSONB | NULL | Additional metadata (capabilities, config) |
| `created_at` | TIMESTAMP | NOT NULL | Record creation time |
| `updated_at` | TIMESTAMP | NOT NULL | Last update time |

**Indexes**:
- `idx_agents_tenant` ON (`tenant_id`)
- `idx_agents_status` ON (`status`)
- `idx_agents_last_seen` ON (`last_seen_at` DESC)
- `idx_agents_hostname` ON (`tenant_id`, `hostname`) UNIQUE

**JSON Example**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "660e8400-e29b-41d4-a716-446655440000",
  "hostname": "web-server-01",
  "os": "linux",
  "arch": "amd64",
  "version": "1.2.3",
  "status": "online",
  "last_seen_at": "2025-11-23T12:30:00Z",
  "metadata": {
    "restic_version": "0.16.2",
    "plugins": ["mysql", "postgres"],
    "max_concurrent_tasks": 2
  },
  "created_at": "2025-11-20T10:00:00Z",
  "updated_at": "2025-11-23T12:30:00Z"
}
```

---

##### 2. Policy Model

**Purpose**: Defines backup policies including schedule, retention, and repository configuration.

**Table**: `policies`

**Fields**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique policy identifier |
| `tenant_id` | UUID | FOREIGN KEY, NOT NULL | Multi-tenant isolation |
| `name` | VARCHAR(255) | NOT NULL | Policy name (e.g., "Daily Production Backup") |
| `schedule` | VARCHAR(255) | NOT NULL | Cron expression (e.g., "0 2 * * *") |
| `include_paths` | JSONB | NOT NULL | Array of paths to backup |
| `exclude_paths` | JSONB | NULL | Array of exclusion patterns |
| `repository_url` | TEXT | NOT NULL | Restic repository URL |
| `repository_type` | VARCHAR(50) | NOT NULL | Backend type (s3, sftp, local, rest) |
| `repository_config` | JSONB | NULL | Backend-specific configuration |
| `retention_rules` | JSONB | NOT NULL | Retention policy (keep_daily, etc.) |
| `enabled` | BOOLEAN | NOT NULL, DEFAULT TRUE | Policy active status |
| `created_at` | TIMESTAMP | NOT NULL | Record creation time |
| `updated_at` | TIMESTAMP | NOT NULL | Last update time |

**Indexes**:
- `idx_policies_tenant` ON (`tenant_id`)
- `idx_policies_enabled` ON (`enabled`)
- `idx_policies_name` ON (`tenant_id`, `name`) UNIQUE

**JSON Example**:
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "660e8400-e29b-41d4-a716-446655440000",
  "name": "Daily Production Backup",
  "schedule": "0 2 * * *",
  "include_paths": ["/home", "/etc", "/var/www"],
  "exclude_paths": ["*.log", ".cache", "node_modules"],
  "repository_url": "s3:s3.amazonaws.com/my-backups",
  "repository_type": "s3",
  "repository_config": {
    "bucket": "my-backups",
    "region": "us-east-1",
    "endpoint": "s3.amazonaws.com"
  },
  "retention_rules": {
    "keep_last": 5,
    "keep_daily": 7,
    "keep_weekly": 4,
    "keep_monthly": 12,
    "keep_yearly": 3
  },
  "enabled": true,
  "created_at": "2025-11-20T10:00:00Z",
  "updated_at": "2025-11-22T14:30:00Z"
}
```

---

##### 3. AgentPolicyLink Model

**Purpose**: Many-to-many relationship between agents and policies. An agent can have multiple policies, and a policy can be assigned to multiple agents.

**Table**: `agent_policy_links`

**Fields**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `agent_id` | UUID | FOREIGN KEY, NOT NULL | Reference to agent |
| `policy_id` | UUID | FOREIGN KEY, NOT NULL | Reference to policy |
| `created_at` | TIMESTAMP | NOT NULL | Link creation time |

**Indexes**:
- `idx_agent_policy_links_agent` ON (`agent_id`)
- `idx_agent_policy_links_policy` ON (`policy_id`)
- PRIMARY KEY (`agent_id`, `policy_id`)

**JSON Example**:
```json
{
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "policy_id": "770e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-11-20T10:05:00Z"
}
```

---

##### 4. BackupRun Model

**Purpose**: Records the execution of a backup task, including start/end times, status, statistics, and logs.

**Table**: `backup_runs`

**Fields**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique run identifier |
| `tenant_id` | UUID | FOREIGN KEY, NOT NULL | Multi-tenant isolation |
| `agent_id` | UUID | FOREIGN KEY, NOT NULL | Agent that executed backup |
| `policy_id` | UUID | FOREIGN KEY, NOT NULL | Policy that defined backup |
| `target_id` | UUID | FOREIGN KEY, NULL | Target (legacy compatibility) |
| `start_time` | TIMESTAMP | NOT NULL | Backup start time |
| `end_time` | TIMESTAMP | NULL | Backup end time (NULL if running) |
| `status` | VARCHAR(50) | NOT NULL | Status (running, success, failed) |
| `error_message` | TEXT | NULL | Error message if failed |
| `files_new` | INTEGER | NULL | Count of new files backed up |
| `files_changed` | INTEGER | NULL | Count of changed files |
| `files_unmodified` | INTEGER | NULL | Count of unmodified files |
| `dirs_new` | INTEGER | NULL | Count of new directories |
| `dirs_changed` | INTEGER | NULL | Count of changed directories |
| `dirs_unmodified` | INTEGER | NULL | Count of unmodified directories |
| `data_added` | BIGINT | NULL | Bytes added to repository |
| `total_files_processed` | BIGINT | NULL | Total files processed |
| `total_bytes_processed` | BIGINT | NULL | Total bytes processed |
| `duration_seconds` | FLOAT | NULL | Total duration in seconds |
| `snapshot_id` | VARCHAR(255) | NULL | Restic snapshot ID if successful |
| `created_at` | TIMESTAMP | NOT NULL | Record creation time |
| `updated_at` | TIMESTAMP | NOT NULL | Last update time |

**Indexes**:
- `idx_backup_runs_tenant` ON (`tenant_id`)
- `idx_backup_runs_agent` ON (`agent_id`)
- `idx_backup_runs_policy` ON (`policy_id`)
- `idx_backup_runs_status` ON (`status`)
- `idx_backup_runs_start_time` ON (`start_time` DESC)
- `idx_backup_runs_snapshot` ON (`snapshot_id`)

**JSON Example**:
```json
{
  "id": "880e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "660e8400-e29b-41d4-a716-446655440000",
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "policy_id": "770e8400-e29b-41d4-a716-446655440000",
  "target_id": null,
  "start_time": "2025-11-23T02:00:00Z",
  "end_time": "2025-11-23T02:15:30Z",
  "status": "success",
  "error_message": null,
  "files_new": 1234,
  "files_changed": 567,
  "files_unmodified": 89012,
  "dirs_new": 45,
  "dirs_changed": 23,
  "dirs_unmodified": 678,
  "data_added": 5368709120,
  "total_files_processed": 90813,
  "total_bytes_processed": 107374182400,
  "duration_seconds": 930.5,
  "snapshot_id": "a1b2c3d4",
  "created_at": "2025-11-23T02:00:00Z",
  "updated_at": "2025-11-23T02:15:30Z"
}
```

---

##### 5. BackupRunLog Model (Supporting Table)

**Purpose**: Stores individual log entries for backup runs, enabling real-time log streaming and post-run analysis.

**Table**: `backup_run_logs`

**Fields**:

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY | Unique log entry identifier |
| `backup_run_id` | UUID | FOREIGN KEY, NOT NULL | Reference to backup run |
| `timestamp` | TIMESTAMP | NOT NULL | Log entry timestamp |
| `level` | VARCHAR(20) | NOT NULL | Log level (info, warn, error) |
| `message` | TEXT | NOT NULL | Log message content |
| `created_at` | TIMESTAMP | NOT NULL | Record creation time |

**Indexes**:
- `idx_backup_run_logs_run` ON (`backup_run_id`)
- `idx_backup_run_logs_timestamp` ON (`timestamp` DESC)
- `idx_backup_run_logs_level` ON (`level`)

**JSON Example**:
```json
{
  "id": "990e8400-e29b-41d4-a716-446655440000",
  "backup_run_id": "880e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-23T02:05:23Z",
  "level": "info",
  "message": "processed 10000 files, 5.2 GB",
  "created_at": "2025-11-23T02:05:23Z"
}
```

---

#### Relationship Cardinalities

**One-to-Many Relationships**:
- `Tenant` → `Agent` (1:N) - A tenant has many agents
- `Tenant` → `Policy` (1:N) - A tenant has many policies
- `Agent` → `BackupRun` (1:N) - An agent executes many backup runs
- `Policy` → `BackupRun` (1:N) - A policy defines many backup runs
- `BackupRun` → `BackupRunLog` (1:N) - A backup run has many log entries
- `BackupRun` → `Snapshot` (1:N) - A backup run can create multiple snapshots (one per source path)

**Many-to-Many Relationships**:
- `Agent` ←→ `Policy` (M:N via `AgentPolicyLink`) - An agent can have multiple policies, a policy can be assigned to multiple agents

**Optional Relationships**:
- `Target` → `Agent` (N:1, optional) - A target may be managed by an agent (NULL for direct-poll mode)
- `Target` → `Policy` (N:1, optional) - A target may follow a policy (NULL for manual management)

---

#### Database Constraints

**Foreign Key Constraints**:
```sql
ALTER TABLE agents
    ADD CONSTRAINT fk_agents_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

ALTER TABLE policies
    ADD CONSTRAINT fk_policies_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

ALTER TABLE agent_policy_links
    ADD CONSTRAINT fk_agent_policy_links_agent
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE;

ALTER TABLE agent_policy_links
    ADD CONSTRAINT fk_agent_policy_links_policy
    FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE;

ALTER TABLE backup_runs
    ADD CONSTRAINT fk_backup_runs_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

ALTER TABLE backup_runs
    ADD CONSTRAINT fk_backup_runs_agent
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE RESTRICT;

ALTER TABLE backup_runs
    ADD CONSTRAINT fk_backup_runs_policy
    FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE RESTRICT;

ALTER TABLE backup_run_logs
    ADD CONSTRAINT fk_backup_run_logs_run
    FOREIGN KEY (backup_run_id) REFERENCES backup_runs(id) ON DELETE CASCADE;
```

**Check Constraints**:
```sql
ALTER TABLE agents
    ADD CONSTRAINT chk_agents_status
    CHECK (status IN ('online', 'offline', 'error'));

ALTER TABLE agents
    ADD CONSTRAINT chk_agents_os
    CHECK (os IN ('linux', 'windows', 'darwin'));

ALTER TABLE backup_runs
    ADD CONSTRAINT chk_backup_runs_status
    CHECK (status IN ('created', 'running', 'success', 'failed', 'cancelled'));

ALTER TABLE backup_run_logs
    ADD CONSTRAINT chk_backup_run_logs_level
    CHECK (level IN ('debug', 'info', 'warn', 'error'));
```

**Unique Constraints**:
```sql
-- Unique hostname per tenant
ALTER TABLE agents
    ADD CONSTRAINT uk_agents_hostname
    UNIQUE (tenant_id, hostname);

-- Unique policy name per tenant
ALTER TABLE policies
    ADD CONSTRAINT uk_policies_name
    UNIQUE (tenant_id, name);

-- Agent-Policy link is unique
ALTER TABLE agent_policy_links
    ADD CONSTRAINT pk_agent_policy_links
    PRIMARY KEY (agent_id, policy_id);
```

---

### Full Schema DDL (PostgreSQL)

```sql
-- Managed by AE Backend base module
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Restic module entities
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hostname VARCHAR(255) NOT NULL,
    os VARCHAR(50) NOT NULL CHECK (os IN ('linux', 'windows', 'darwin')),
    arch VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('online', 'offline', 'error')),
    last_seen_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_agents_hostname UNIQUE (tenant_id, hostname)
);

CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    schedule VARCHAR(255) NOT NULL,
    include_paths JSONB NOT NULL,
    exclude_paths JSONB,
    repository_url TEXT NOT NULL,
    repository_type VARCHAR(50) NOT NULL,
    repository_config JSONB,
    retention_rules JSONB NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_policies_name UNIQUE (tenant_id, name)
);

CREATE TABLE agent_policy_links (
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (agent_id, policy_id)
);

CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    password_file TEXT,
    cert_file TEXT,
    environment_vars JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('direct', 'agent')),
    repository_id UUID REFERENCES repositories(id) ON DELETE RESTRICT,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    policy_id UUID REFERENCES policies(id) ON DELETE SET NULL,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_targets_name UNIQUE (tenant_id, name)
);

CREATE TABLE backup_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE RESTRICT,
    target_id UUID REFERENCES targets(id) ON DELETE SET NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    status VARCHAR(50) NOT NULL CHECK (status IN ('created', 'running', 'success', 'failed', 'cancelled')),
    error_message TEXT,
    files_new INTEGER,
    files_changed INTEGER,
    files_unmodified INTEGER,
    dirs_new INTEGER,
    dirs_changed INTEGER,
    dirs_unmodified INTEGER,
    data_added BIGINT,
    total_files_processed BIGINT,
    total_bytes_processed BIGINT,
    duration_seconds FLOAT,
    snapshot_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE backup_run_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_run_id UUID NOT NULL REFERENCES backup_runs(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    level VARCHAR(20) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    target_id UUID REFERENCES targets(id) ON DELETE CASCADE,
    backup_run_id UUID REFERENCES backup_runs(id) ON DELETE SET NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    short_id VARCHAR(10),
    time TIMESTAMP NOT NULL,
    hostname VARCHAR(255),
    username VARCHAR(255),
    paths JSONB,
    tags JSONB,
    parent VARCHAR(255),
    tree VARCHAR(255),
    source VARCHAR(50),
    file_count INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    healthy BOOLEAN NOT NULL,
    error_message TEXT,
    snapshot_age_hours INTEGER,
    checked_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_agents_tenant ON agents(tenant_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_last_seen ON agents(last_seen_at DESC);

CREATE INDEX idx_policies_tenant ON policies(tenant_id);
CREATE INDEX idx_policies_enabled ON policies(enabled);

CREATE INDEX idx_agent_policy_links_agent ON agent_policy_links(agent_id);
CREATE INDEX idx_agent_policy_links_policy ON agent_policy_links(policy_id);

CREATE INDEX idx_targets_tenant ON targets(tenant_id);
CREATE INDEX idx_targets_mode ON targets(mode);
CREATE INDEX idx_targets_agent ON targets(agent_id);
CREATE INDEX idx_targets_policy ON targets(policy_id);

CREATE INDEX idx_backup_runs_tenant ON backup_runs(tenant_id);
CREATE INDEX idx_backup_runs_agent ON backup_runs(agent_id);
CREATE INDEX idx_backup_runs_policy ON backup_runs(policy_id);
CREATE INDEX idx_backup_runs_status ON backup_runs(status);
CREATE INDEX idx_backup_runs_start_time ON backup_runs(start_time DESC);
CREATE INDEX idx_backup_runs_snapshot ON backup_runs(snapshot_id);

CREATE INDEX idx_backup_run_logs_run ON backup_run_logs(backup_run_id);
CREATE INDEX idx_backup_run_logs_timestamp ON backup_run_logs(timestamp DESC);
CREATE INDEX idx_backup_run_logs_level ON backup_run_logs(level);

CREATE INDEX idx_snapshots_target ON snapshots(target_id);
CREATE INDEX idx_snapshots_time ON snapshots(time DESC);
CREATE INDEX idx_snapshots_run ON snapshots(backup_run_id);

CREATE INDEX idx_health_checks_target ON health_checks(target_id);
CREATE INDEX idx_health_checks_checked_at ON health_checks(checked_at DESC);
```

### Database Migrations

The system uses a versioned migration system to manage schema changes and ensure backward compatibility during upgrades.

#### Migration Architecture

```
internal/store/
├── migrations.go        # Migration runner and version definitions
├── migrations_test.go   # Migration tests (TDD)
└── models.go           # GORM models

cmd/migrate/
└── main.go             # CLI tool for running migrations
```

#### Migration Runner

The `MigrationRunner` provides a framework for managing database schema changes:

- **Version Tracking**: `schema_migrations` table tracks applied migrations
- **Idempotency**: Migrations can be run multiple times safely
- **Transactional**: Each migration runs in a transaction (rollback on failure)
- **Ordered Execution**: Migrations run in version order (001, 002, ...)

**Schema Migrations Table**:
```sql
CREATE TABLE schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    description VARCHAR(255),
    applied_at TIMESTAMP NOT NULL
);
```

#### Migration 001: v0.x to v1.x

**Version**: `001`  
**Description**: Create agent, policy, and backup run tables; migrate from v0.x schema

This migration performs the following operations:

1. **Create v1.x Tables**: 
   - `agents` - Backup agents running on target machines
   - `policies` - Backup policies with schedules and retention rules
   - `agent_policy_links` - Many-to-many relationship
   - `backup_runs` - Execution records of backup tasks
   - `backup_run_logs` - Log streaming for backup runs

2. **Data Migration from v0.x**:
   - Converts `targets` table entries to `agents` + `policies`
   - Preserves retention settings (`keep_last`, `keep_daily`, etc.)
   - Detects repository type from URL (s3, sftp, local, rest)
   - Links agents to policies via `agent_policy_links`
   - Maintains backward compatibility with legacy tables

3. **Default Values**:
   - Agent OS/Arch: `"unknown"` (not available in v0 schema)
   - Agent Version: `"legacy"`
   - Agent Status: `"offline"`
   - Policy Schedule: `"0 2 * * *"` (daily at 2am)
   - Metadata: Stores original `target_id` for traceability

**Example Migration Data Transformation**:

v0.x Target:
```json
{
  "id": 42,
  "name": "web-server-01",
  "repository": "s3:bucket/path",
  "keep_daily": 7,
  "keep_weekly": 4,
  "disabled": false
}
```

v1.x Agent + Policy:
```json
{
  "agent": {
    "id": "uuid-generated",
    "tenant_id": "default-tenant-id",
    "hostname": "web-server-01",
    "os": "unknown",
    "status": "offline",
    "metadata": {"migrated_from": "v0_target", "target_id": 42}
  },
  "policy": {
    "id": "uuid-generated",
    "tenant_id": "default-tenant-id",
    "name": "web-server-01 Policy",
    "schedule": "0 2 * * *",
    "repository_url": "s3:bucket/path",
    "repository_type": "s3",
    "retention_rules": {"keep_daily": 7, "keep_weekly": 4},
    "enabled": true
  }
}
```

#### Running Migrations

**Automatic (on server start)**:
```go
store, err := store.New("data/restic-monitor.db")
// Migrations run automatically
```

**Manual (CLI tool)**:
```bash
# Run migrations on existing database
go run cmd/migrate/main.go -db data/restic-monitor.db

# Specify tenant ID
go run cmd/migrate/main.go -db data/restic-monitor.db -tenant "uuid-here"

# Test with in-memory database
go run cmd/migrate/main.go -db ":memory:"
```

**Output Example**:
```
Running migrations on database: data/restic-monitor.db
Tenant ID: 534a7cd8-e804-42b6-a918-88aa96704b1d

Found 1 migrations
→ Running migration 001: Create agent, policy, and backup run tables; migrate from v0.x schema
✓ Migration 001 completed

All migrations completed successfully!

Applied migrations:
  001 - Create agent, policy, and backup run tables; migrate from v0.x schema (applied at 2025-11-24 12:33:41)
```

#### Adding New Migrations

To add a new migration:

1. **Create migration function** in `internal/store/migrations.go`:
```go
func GetMigration002Example(tenantID uuid.UUID) Migration {
    return Migration{
        Version:     "002",
        Description: "Add new feature",
        Up: func(tx *gorm.DB) error {
            // SQL or GORM operations
            return nil
        },
    }
}
```

2. **Add to migration list**:
```go
func GetAllMigrations(defaultTenantID uuid.UUID) []Migration {
    return []Migration{
        GetMigration001CreateAgentTables(defaultTenantID),
        GetMigration002Example(defaultTenantID),  // Add here
    }
}
```

3. **Write tests** in `internal/store/migrations_test.go`:
```go
func TestMigration002(t *testing.T) {
    // Test migration logic
}
```

4. **Test the migration**:
```bash
go test ./internal/store/... -run TestMigration002
```

#### Migration Best Practices

1. **Idempotency**: Migrations must be safe to run multiple times
2. **Transactions**: Use transactions to ensure atomicity
3. **Data Preservation**: Never delete user data during migration
4. **Testing**: Write comprehensive tests before production deployment
5. **Versioning**: Use sequential numeric versions (001, 002, 003...)
6. **Documentation**: Document schema changes in this file
7. **Backward Compatibility**: Keep legacy tables during transition period

---

## API Documentation

This section documents all REST API endpoints exposed by the restic-monitor orchestrator.

### Agent APIs

Agent APIs enable backup agents to communicate with the orchestrator for registration, task polling, and status reporting.

#### Agent Registration

**Endpoint**: `POST /agents/register`

Allows agents to register themselves with the orchestrator and provide machine metadata.

**Full Documentation**: [Agent Registration API](api/agent-registration.md)

**Key Features**:
- Automatic agent discovery and registration
- Metadata updates on re-registration
- Idempotent operation (prevents duplicates)
- Uses existing authentication mechanism

**Request Example**:
```json
{
  "hostname": "web-server-01.example.com",
  "os": "linux",
  "arch": "amd64", 
  "version": "1.0.0",
  "ip": "192.168.1.100"
}
```

**Response Example**:
```json
{
  "agentId": "a3f7c8e9-4d2a-4b5c-9e8f-7d6c5b4a3f2e",
  "hostname": "web-server-01.example.com",
  "registeredAt": "2025-11-24T12:00:00Z",
  "updatedAt": "2025-11-24T12:00:00Z",
  "message": "Agent registered successfully"
}
```

#### Agent Heartbeat

**Endpoint**: `POST /agents/{id}/heartbeat`

Allows agents to send periodic health and status updates to the orchestrator.

**Full Documentation**: [Agent Heartbeat API](api/agent-heartbeat.md)

**Key Features**:
- Real-time agent health monitoring
- Automatic online/offline detection
- Disk usage tracking
- Backup status reporting
- Configurable heartbeat intervals

**Request Example**:
```json
{
  "version": "1.2.3",
  "os": "linux",
  "uptimeSeconds": 864000,
  "disks": [
    {
      "mountPath": "/",
      "freeBytes": 50000000000,
      "totalBytes": 100000000000
    }
  ],
  "lastBackupStatus": "success"
}
```

**Response Example**:
```json
{
  "status": "ok",
  "nextTaskCheckAfterSeconds": 30
}
```

#### Future Agent APIs

The following agent APIs are planned for future implementation:

- **Task Polling** (`GET /agents/{id}/tasks`) - Poll for assigned backup tasks
- **Task Status Update** (`POST /agents/{id}/tasks/{taskId}/status`) - Report task execution status
- **Log Streaming** (`POST /agents/{id}/tasks/{taskId}/logs`) - Stream backup logs

### Orchestrator APIs

Orchestrator APIs enable operators to manage agents, policies, and backup runs through the web UI or CLI tools.

#### Agent Management

- `GET /api/agents` - List all registered agents
- `GET /api/agents/{id}` - Get agent details
- `PUT /api/agents/{id}` - Update agent configuration
- `DELETE /api/agents/{id}` - Deregister agent

#### Policy Management

- `GET /api/policies` - List all backup policies
- `GET /api/policies/{id}` - Get policy details
- `POST /api/policies` - Create new policy
- `PUT /api/policies/{id}` - Update policy
- `DELETE /api/policies/{id}` - Delete policy

#### Backup Run Management

- `GET /api/backup-runs` - List backup run history
- `GET /api/backup-runs/{id}` - Get backup run details
- `GET /api/backup-runs/{id}/logs` - Get backup run logs
- `POST /api/backup-runs/{id}/cancel` - Cancel running backup

### API Standards

All APIs follow these conventions:

- **Content Type**: `application/json`
- **Authentication**: Bearer token or Basic Auth (existing mechanism)
- **Timestamps**: RFC 3339 format (e.g., `2025-11-24T12:00:00Z`)
- **UUIDs**: RFC 4122 format with hyphens
- **Error Responses**: Consistent error structure with `error` and optional `details` fields
- **HTTP Status Codes**: Standard REST conventions (200, 201, 400, 401, 404, 500)

### OpenAPI/Swagger

Full OpenAPI 3.0 specification available at:
- **Development**: `http://localhost:8080/swagger/`
- **API Spec**: `api/swagger.json`
- **Generated Docs**: `api/docs.go`

---

## Security Model
-- Managed by base module
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Restic module entities
CREATE TABLE restic_targets (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(20) NOT NULL, -- 'direct' or 'agent'
    repository_id UUID REFERENCES restic_repositories(id),
    agent_id UUID REFERENCES restic_agents(id) NULL,
    policy_id UUID REFERENCES restic_policies(id) NULL,
    disabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE restic_repositories (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    type VARCHAR(50) NOT NULL, -- 's3', 'sftp', 'local', etc.
    url TEXT NOT NULL,
    password_file TEXT,
    cert_file TEXT,
    environment_vars JSONB, -- Backend-specific config
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE restic_agents (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    hostname VARCHAR(255) NOT NULL,
    version VARCHAR(50),
    token_hash VARCHAR(255), -- Hashed authentication token
    status VARCHAR(50), -- 'online', 'offline', 'error'
    last_heartbeat TIMESTAMP,
    metadata JSONB, -- OS, arch, capabilities
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE restic_snapshots (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    target_id UUID REFERENCES restic_targets(id),
    snapshot_id VARCHAR(255) NOT NULL,
    short_id VARCHAR(10),
    time TIMESTAMP NOT NULL,
    hostname VARCHAR(255),
    username VARCHAR(255),
    paths JSONB, -- Array of paths
    tags JSONB, -- Array of tags
    parent VARCHAR(255),
    tree VARCHAR(255),
    source VARCHAR(50), -- 'direct_poll' or 'agent_report'
    file_count INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE restic_tasks (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    agent_id UUID REFERENCES restic_agents(id),
    target_id UUID REFERENCES restic_targets(id),
    type VARCHAR(50) NOT NULL, -- 'backup', 'check', 'prune'
    status VARCHAR(50) NOT NULL, -- 'created', 'queued', 'running', 'success', 'failed'
    parameters JSONB,
    result JSONB, -- Snapshot ID, errors, statistics
    logs TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE restic_policies (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    schedule VARCHAR(255), -- Cron expression
    retention JSONB, -- keep_daily, keep_weekly, etc.
    sources JSONB, -- Paths to backup
    exclude JSONB, -- Exclusion patterns
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE restic_health_checks (
    id UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    target_id UUID REFERENCES restic_targets(id),
    healthy BOOLEAN NOT NULL,
    error_message TEXT,
    snapshot_age_hours INTEGER,
    checked_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_snapshots_target ON restic_snapshots(target_id);
CREATE INDEX idx_snapshots_time ON restic_snapshots(time DESC);
CREATE INDEX idx_tasks_agent ON restic_tasks(agent_id, status);
CREATE INDEX idx_tasks_status ON restic_tasks(status) WHERE status IN ('queued', 'running');
CREATE INDEX idx_agents_heartbeat ON restic_agents(last_heartbeat DESC);
```

---

## Security Model

### Authentication & Authorization

#### Orchestrator Security (AE Backend)

**User Authentication:**
- JWT tokens issued by base module
- Role-based access control (RBAC)
- Tenant isolation

**Roles:**
- `admin` - Full access to all restic resources
- `operator` - Create/manage backups and policies
- `viewer` - Read-only access to status and logs

**Example:**
```go
// Middleware enforces tenant isolation
func (h *PolicyHandler) GetPolicies(c *gin.Context) {
    tenantID := c.GetString("tenant_id") // From JWT
    policies, _ := h.policyService.GetByTenant(tenantID)
    c.JSON(200, policies)
}
```

#### Agent Security

**Agent Authentication:**
- Unique token per agent (generated at registration)
- Token stored hashed in database (bcrypt)
- Token sent in `Authorization: Bearer <token>` header

**Agent Registration Flow:**
```
1. Admin creates registration token (one-time use, expires 1 hour)
2. Agent POSTs to /agents/register with token
3. Orchestrator validates token, creates agent record
4. Orchestrator generates permanent agent token
5. Agent stores token securely (OS keyring)
6. Registration token is revoked
```

**Token Rotation:**
```go
// Agent requests new token before expiry
func (c *OrchestratorClient) RotateToken() (string, error)

// Orchestrator issues new token, keeps old valid for 5 minutes
func (m *AgentManager) RotateAgentToken(agentID string) (string, error)
```

### Transport Security

**TLS Everywhere:**
- Orchestrator ↔ Agent: HTTPS with TLS 1.3
- Orchestrator ↔ Restic Repos: TLS certificates (configurable)
- Agent ↔ Restic Repos: TLS certificates (configurable)

**Certificate Management:**
```yaml
# Orchestrator config
tls:
  cert_file: /etc/restic-monitor/tls.crt
  key_file: /etc/restic-monitor/tls.key
  
# Agent config
orchestrator:
  url: https://restic-monitor.example.com
  ca_cert: /etc/restic-agent/ca.crt # Verify orchestrator
```

### Repository Security

**Restic Encryption:**
- All data encrypted at rest (Restic native)
- Repository password required for all operations
- Password stored in secure file (chmod 600)

**Password Management:**
```yaml
# Agent config
repository:
  password_file: /etc/restic/repo-password
  
# Orchestrator stores password reference, not content
# Actual password file deployed via configuration management
```

**Backend Credentials:**
- S3 keys, SSH keys, Azure keys stored in files
- Files referenced in configuration, not embedded
- Secrets deployed via Vault, AWS Secrets Manager, etc.

### Tenant Isolation (Multi-Tenant)

**Database Level:**
- All queries filtered by `tenant_id`
- Row-level security (PostgreSQL RLS)

```sql
-- Enable RLS on all restic tables
ALTER TABLE restic_targets ENABLE ROW LEVEL SECURITY;

-- Policy: Users see only their tenant's data
CREATE POLICY tenant_isolation ON restic_targets
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Application Level:**
```go
// Middleware extracts tenant from JWT
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := extractClaims(c)
        c.Set("tenant_id", claims.TenantID)
        
        // Set for database queries
        db.Exec("SET app.current_tenant = ?", claims.TenantID)
        c.Next()
    }
}
```

---

## Deployment Architecture

### Orchestrator Deployment

**Container (Docker/Kubernetes):**

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ae-backend-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/ae-backend-server /usr/local/bin/
COPY frontend/dist /app/static

CMD ["ae-backend-server"]
```

**Docker Compose:**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ae_backend
      POSTGRES_USER: ae_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  ae-backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: ae_backend
      DB_USER: ae_user
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      - postgres
    volumes:
      - ./config:/etc/restic-monitor

volumes:
  postgres_data:
```

**Kubernetes:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ae-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ae-backend
  template:
    metadata:
      labels:
        app: ae-backend
    spec:
      containers:
      - name: server
        image: ae-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres-service
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: ae-secrets
              key: jwt-secret
---
apiVersion: v1
kind: Service
metadata:
  name: ae-backend-service
spec:
  selector:
    app: ae-backend
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Agent Deployment

**Systemd (Linux):**

```ini
# /etc/systemd/system/restic-agent.service
[Unit]
Description=Restic Backup Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=restic
Group=restic
ExecStart=/usr/local/bin/restic-agent --config /etc/restic-agent/config.yaml
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

**Windows Service:**

```powershell
# Install as Windows service
New-Service -Name "ResticAgent" `
    -BinaryPathName "C:\Program Files\ResticAgent\restic-agent.exe --config C:\ProgramData\ResticAgent\config.yaml" `
    -DisplayName "Restic Backup Agent" `
    -StartupType Automatic

Start-Service ResticAgent
```

**macOS (launchd):**

```xml
<!-- ~/Library/LaunchAgents/com.restic-monitor.agent.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.restic-monitor.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/restic-agent</string>
        <string>--config</string>
        <string>/etc/restic-agent/config.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

### High Availability

**Orchestrator HA:**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ AE Backend  │     │ AE Backend  │     │ AE Backend  │
│ Instance 1  │     │ Instance 2  │     │ Instance 3  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                   ┌───────▼───────┐
                   │ Load Balancer │
                   │ (nginx/HAProxy│
                   └───────┬───────┘
                           │
                      HTTPS/WSS
                           │
                   ┌───────▼───────┐
                   │   Agents      │
                   └───────────────┘

       ┌───────────────────┼───────────────────┐
       │                   │                   │
┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐
│ PostgreSQL  │───>│ PostgreSQL  │───>│ PostgreSQL  │
│  Primary    │    │  Standby 1  │    │  Standby 2  │
└─────────────┘    └─────────────┘    └─────────────┘
  (Streaming Replication)
```

**Agent Resilience:**

- Agents reconnect automatically on network failure
- Task state preserved during connection loss
- Exponential backoff on connection errors
- Local task queue for offline operation

---

## Component Architecture

This section provides detailed breakdowns of each system component, their responsibilities, dependencies, API surfaces, and error conditions.

### 1. Orchestrator Backend

The orchestrator backend is the central management system that coordinates all backup operations, monitors health, and provides the API for the frontend and agents.

#### 1.1 API Server Component

**Responsibilities:**
- Expose RESTful API endpoints for frontend and agents
- Handle HTTP request routing and middleware
- Authenticate and authorize requests (JWT validation)
- Enforce tenant isolation for multi-tenant deployments
- Generate OpenAPI/Swagger documentation
- Serve static frontend assets
- Manage WebSocket connections for real-time updates

**Dependencies:**
- **External:**
  - Gin HTTP framework (routing, middleware)
  - AE Backend bootstrap system (module initialization)
  - PostgreSQL database (via GORM)
  - JWT library (token validation)
- **Internal:**
  - MonitorService (health checks)
  - AgentManager (agent lifecycle)
  - TaskService (task management)
  - PolicyEngine (policy evaluation)

**API Surface:**

```go
type APIServer struct {
    router      *gin.Engine
    config      *config.Config
    db          *gorm.DB
    logger      core.Logger
    eventBus    core.EventBus
}

// Initialize sets up routes and middleware
func (s *APIServer) Initialize() error

// Start begins listening on configured port
func (s *APIServer) Start(ctx context.Context) error

// Stop gracefully shuts down the server
func (s *APIServer) Stop(ctx context.Context) error

// RegisterRoutes adds module routes to router
func (s *APIServer) RegisterRoutes(provider core.RouteProvider) error
```

**Endpoints:**

*Current (v0.x):*
- `GET /api/v1/restic/status` - List all targets
- `GET /api/v1/restic/status/:name` - Get target details
- `GET /api/v1/restic/status/:name/snapshots` - List snapshots
- `GET /api/v1/restic/status/:name/snapshots/:id/files` - Get file list
- `GET /api/v1/health` - Health check

*Future (v1.x+):*
- `POST /api/v1/restic/agents/register` - Agent registration
- `POST /api/v1/restic/agents/:id/heartbeat` - Heartbeat
- `GET /api/v1/restic/agents/:id/tasks` - Poll tasks (long-polling)
- `POST /api/v1/restic/tasks/:id/status` - Update task status
- `POST /api/v1/restic/tasks/:id/logs` - Stream logs
- `GET /api/v1/restic/policies` - List policies
- `POST /api/v1/restic/policies` - Create policy

**Error Conditions:**

| Error | HTTP Code | Description | Recovery |
|-------|-----------|-------------|----------|
| `InvalidToken` | 401 | JWT token invalid/expired | Client must re-authenticate |
| `InsufficientPermissions` | 403 | User lacks required role | Contact admin for permissions |
| `TargetNotFound` | 404 | Target name doesn't exist | Check target configuration |
| `AgentOffline` | 503 | Agent not responding | Wait for agent reconnection |
| `DatabaseError` | 500 | Database connection lost | Check DB connection, retry |
| `ValidationError` | 400 | Invalid request parameters | Fix request body/params |
| `RateLimitExceeded` | 429 | Too many requests | Implement exponential backoff |

**Configuration:**

```go
type APIConfig struct {
    Host            string        // Listen address (default: 0.0.0.0)
    Port            string        // Listen port (default: 8080)
    ReadTimeout     time.Duration // Request read timeout (default: 30s)
    WriteTimeout    time.Duration // Response write timeout (default: 30s)
    MaxHeaderBytes  int           // Max header size (default: 1MB)
    EnableCORS      bool          // Enable CORS middleware
    EnableSwagger   bool          // Enable Swagger UI
    TrustedProxies  []string      // Trusted proxy IPs
}
```

---

#### 1.2 Monitor Service Component

**Responsibilities:**
- Poll Restic repositories on scheduled intervals (current direct-poll mode)
- Execute `restic snapshots` command to retrieve backup history
- Execute `restic check` command to verify repository health
- Execute `restic stats` command to get snapshot file counts
- Parse Restic JSON output and store in database
- Detect health status changes and emit events
- Calculate snapshot age and flag aging backups
- Support multiple repository backends (S3, SFTP, local, etc.)
- Handle TLS certificate configuration for REST backends

**Dependencies:**
- **External:**
  - Restic binary (CLI execution)
  - Repository storage (S3, SFTP, filesystem, etc.)
  - TLS certificates (for REST repositories)
- **Internal:**
  - Database (GORM models: Target, Snapshot, HealthCheck)
  - EventBus (emit HealthChanged, SnapshotDiscovered events)
  - Logger (structured logging)

**API Surface:**

```go
type MonitorService struct {
    db          *gorm.DB
    logger      core.Logger
    eventBus    core.EventBus
    resticPath  string
    targets     []config.Target
    interval    time.Duration
    stopChan    chan struct{}
    wg          sync.WaitGroup
}

// StartMonitoring begins periodic polling of all targets
func (s *MonitorService) StartMonitoring(ctx context.Context) error

// StopMonitoring gracefully stops all polling loops
func (s *MonitorService) StopMonitoring(ctx context.Context) error

// CheckTarget polls a single target immediately
func (s *MonitorService) CheckTarget(ctx context.Context, target config.Target) error

// GetSnapshots retrieves snapshots for a target from database
func (s *MonitorService) GetSnapshots(targetID string, limit int) ([]Snapshot, error)

// GetHealth retrieves latest health check for target
func (s *MonitorService) GetHealth(targetID string) (*HealthCheck, error)

// RefreshTarget forces immediate poll of target
func (s *MonitorService) RefreshTarget(ctx context.Context, targetName string) error

// GetSnapshotFiles retrieves file list for specific snapshot
func (s *MonitorService) GetSnapshotFiles(targetID, snapshotID string) ([]string, error)
```

**Restic Command Integration:**

```go
// Internal helper methods
func (s *MonitorService) execRestic(ctx context.Context, target config.Target, args []string) ([]byte, error)
func (s *MonitorService) parseSnapshots(jsonOutput []byte) ([]Snapshot, error)
func (s *MonitorService) parseStats(jsonOutput []byte) (int, error)
func (s *MonitorService) checkHealth(ctx context.Context, target config.Target) (bool, string, error)
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `ResticNotFound` | Fatal | Restic binary not in PATH | Install Restic, set RESTIC_PATH env |
| `RepositoryLocked` | Transient | Another process has lock | Wait for lock release, retry |
| `RepositoryCorrupted` | Critical | `check` command failed | Manual `restic rebuild-index`, restore from backup |
| `AuthenticationFailed` | Config | Invalid repository password | Check password file, update config |
| `NetworkTimeout` | Transient | Connection to backend timed out | Check network, increase timeout |
| `BackendUnavailable` | Transient | S3/SFTP backend unreachable | Check backend status, retry |
| `CertificateError` | Config | TLS cert invalid/expired | Update RESTIC_CERT_FILE path |
| `OutOfMemory` | Fatal | Restic process OOM | Increase memory limit, reduce cache size |
| `SnapshotTooOld` | Warning | Last snapshot exceeds maxage | Trigger backup task |

**Configuration:**

```go
type MonitorConfig struct {
    Interval       time.Duration // Poll interval (default: 5m)
    Timeout        time.Duration // Command timeout (default: 10m)
    MaxConcurrent  int           // Max parallel checks (default: 5)
    ResticPath     string        // Path to restic binary
    CacheDir       string        // Restic cache directory
    EnableMock     bool          // Use mock data for testing
}
```

**Events Emitted:**

```go
// HealthChanged - Repository health status changed
type HealthChangedEvent struct {
    TargetID    string
    TargetName  string
    OldHealth   bool
    NewHealth   bool
    ErrorMsg    string
    Timestamp   time.Time
}

// SnapshotDiscovered - New snapshot detected
type SnapshotDiscoveredEvent struct {
    TargetID    string
    SnapshotID  string
    Hostname    string
    Paths       []string
    Tags        []string
    Timestamp   time.Time
}
```

---

#### 1.3 Agent Manager Component (Future)

**Responsibilities:**
- Handle agent registration and authentication
- Generate and validate agent tokens (JWT or API keys)
- Track agent health status via heartbeat mechanism
- Maintain agent metadata (hostname, version, capabilities)
- Provide agent discovery and listing
- Revoke compromised or decommissioned agents
- Support agent token rotation
- Enforce agent-to-tenant mapping (multi-tenancy)

**Dependencies:**
- **External:**
  - JWT library (token generation/validation)
  - bcrypt (token hashing)
- **Internal:**
  - Database (Agent entity)
  - EventBus (AgentRegistered, AgentOffline events)
  - Logger

**API Surface:**

```go
type AgentManager struct {
    db          *gorm.DB
    logger      core.Logger
    eventBus    core.EventBus
    tokenSecret string
    tokenTTL    time.Duration
}

// RegisterAgent creates new agent with authentication token
func (m *AgentManager) RegisterAgent(ctx context.Context, req AgentRegistration) (*Agent, string, error)

// AuthenticateAgent validates token and returns agent
func (m *AgentManager) AuthenticateAgent(ctx context.Context, token string) (*Agent, error)

// UpdateHeartbeat records agent health ping
func (m *AgentManager) UpdateHeartbeat(ctx context.Context, agentID string, status HealthStatus) error

// GetAgent retrieves agent by ID
func (m *AgentManager) GetAgent(ctx context.Context, agentID string) (*Agent, error)

// ListAgents returns all agents for tenant
func (m *AgentManager) ListAgents(ctx context.Context, tenantID string) ([]Agent, error)

// RevokeAgent disables agent and invalidates token
func (m *AgentManager) RevokeAgent(ctx context.Context, agentID string) error

// RotateToken generates new token for agent
func (m *AgentManager) RotateToken(ctx context.Context, agentID string) (string, error)

// CheckAgentHealth evaluates if agent is online based on last heartbeat
func (m *AgentManager) CheckAgentHealth(ctx context.Context, agentID string) (bool, error)
```

**Registration Flow:**

```
1. Admin generates one-time registration token (expires in 1h)
2. Agent POSTs to /agents/register with registration token
3. Manager validates registration token
4. Manager creates Agent record in database
5. Manager generates permanent agent token (JWT)
6. Manager revokes registration token
7. Manager returns agent ID + token to agent
8. Agent stores token securely (OS keyring)
```

**Token Format (JWT):**

```json
{
  "sub": "agent-uuid",
  "iss": "restic-monitor",
  "aud": "agent",
  "exp": 1732464000,
  "iat": 1732377600,
  "tenant_id": "tenant-uuid",
  "agent_hostname": "web-server-01",
  "agent_version": "1.0.0"
}
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `InvalidRegistrationToken` | Auth | Registration token invalid/expired | Generate new registration token |
| `AgentAlreadyExists` | Conflict | Hostname already registered | Use different hostname or revoke old |
| `TokenExpired` | Auth | Agent token expired | Agent must re-register or rotate token |
| `TokenRevoked` | Auth | Agent token was revoked | Re-register agent |
| `HeartbeatTimeout` | Warning | No heartbeat for 5+ minutes | Check agent process, network |
| `UnknownAgent` | NotFound | Agent ID doesn't exist | Check agent configuration |

**Configuration:**

```go
type AgentManagerConfig struct {
    TokenSecret          string        // JWT signing secret
    TokenTTL             time.Duration // Token validity (default: 30d)
    RegistrationTokenTTL time.Duration // Registration token TTL (default: 1h)
    HeartbeatTimeout     time.Duration // Consider offline after (default: 5m)
    MaxAgentsPerTenant   int           // Limit agents per tenant (default: 100)
}
```

**Events Emitted:**

```go
// AgentRegistered - New agent joined
type AgentRegisteredEvent struct {
    AgentID     string
    TenantID    string
    Hostname    string
    Version     string
    Timestamp   time.Time
}

// AgentOffline - Agent missed heartbeats
type AgentOfflineEvent struct {
    AgentID        string
    LastHeartbeat  time.Time
    Timestamp      time.Time
}
```

---

#### 1.4 Task Service Component (Future)

**Responsibilities:**
- Create backup/check/prune tasks from policies or manual requests
- Queue tasks and assign to appropriate agents
- Support long-polling for task distribution (pull-based)
- Receive task status updates from agents
- Store task execution logs
- Handle task cancellation
- Retry failed tasks based on policy
- Emit task lifecycle events
- Track task execution statistics

**Dependencies:**
- **External:**
  - None (pure business logic)
- **Internal:**
  - Database (Task entity)
  - EventBus (TaskCreated, TaskCompleted events)
  - AgentManager (validate agent exists)
  - PolicyEngine (evaluate policies)
  - Logger

**API Surface:**

```go
type TaskService struct {
    db          *gorm.DB
    logger      core.Logger
    eventBus    core.EventBus
    agentMgr    *AgentManager
}

// CreateTask generates new task (manual or policy-driven)
func (s *TaskService) CreateTask(ctx context.Context, req TaskRequest) (*Task, error)

// GetPendingTasks retrieves tasks for agent (long-polling support)
func (s *TaskService) GetPendingTasks(ctx context.Context, agentID string, timeout time.Duration) ([]Task, error)

// UpdateTaskStatus records task progress
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus) error

// AppendLogs adds log entries to task
func (s *TaskService) AppendLogs(ctx context.Context, taskID string, logs []LogEntry) error

// GetTask retrieves task by ID
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*Task, error)

// ListTasks returns tasks for target/agent
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]Task, error)

// CancelTask stops running task
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error

// RetryTask creates new task from failed task
func (s *TaskService) RetryTask(ctx context.Context, taskID string) (*Task, error)

// GetTaskStatistics returns execution metrics
func (s *TaskService) GetTaskStatistics(ctx context.Context, agentID string) (*TaskStats, error)
```

**Task Request Structure:**

```go
type TaskRequest struct {
    Type        TaskType              // backup, check, prune
    AgentID     string                // Target agent
    TargetID    string                // Backup target
    Priority    int                   // 0-10, higher = more urgent
    Parameters  map[string]interface{} // Task-specific params
    Metadata    map[string]string     // User-defined tags
}

type TaskType string
const (
    TaskTypeBackup TaskType = "backup"
    TaskTypeCheck  TaskType = "check"
    TaskTypePrune  TaskType = "prune"
    TaskTypeForget TaskType = "forget"
)
```

**Task Status Lifecycle:**

```go
type TaskStatus string
const (
    TaskStatusCreated   TaskStatus = "created"   // Just created
    TaskStatusQueued    TaskStatus = "queued"    // Waiting for agent
    TaskStatusAssigned  TaskStatus = "assigned"  // Agent retrieved task
    TaskStatusRunning   TaskStatus = "running"   // Agent executing
    TaskStatusSuccess   TaskStatus = "success"   // Completed successfully
    TaskStatusFailed    TaskStatus = "failed"    // Error occurred
    TaskStatusCancelled TaskStatus = "cancelled" // User/system cancelled
)
```

**Long-Polling Implementation:**

```go
// GetPendingTasks with long-polling
func (s *TaskService) GetPendingTasks(ctx context.Context, agentID string, timeout time.Duration) ([]Task, error) {
    // Try immediate fetch
    tasks, err := s.fetchPendingTasks(agentID)
    if err != nil || len(tasks) > 0 {
        return tasks, err
    }
    
    // No tasks, wait for new tasks or timeout
    select {
    case <-s.taskAvailable(agentID):
        return s.fetchPendingTasks(agentID)
    case <-time.After(timeout):
        return []Task{}, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `AgentNotFound` | NotFound | Agent ID doesn't exist | Check agent registration |
| `TargetNotFound` | NotFound | Target ID doesn't exist | Check target configuration |
| `InvalidParameters` | Validation | Task params invalid | Fix request parameters |
| `TaskNotFound` | NotFound | Task ID doesn't exist | Check task ID |
| `TaskAlreadyRunning` | Conflict | Can't cancel completed task | N/A |
| `TaskTimeout` | Warning | Task exceeded max duration | Agent may be stuck, retry |
| `AgentOffline` | Unavailable | Agent not responding | Wait for agent reconnection |
| `ConcurrencyExceeded` | RateLimit | Too many concurrent tasks | Wait for tasks to complete |

**Configuration:**

```go
type TaskServiceConfig struct {
    MaxConcurrentPerAgent int           // Max tasks per agent (default: 2)
    TaskTimeout           time.Duration // Max task duration (default: 6h)
    LongPollTimeout       time.Duration // Long-poll duration (default: 30s)
    RetryAttempts         int           // Auto-retry count (default: 3)
    RetryBackoff          time.Duration // Retry delay (default: 5m)
}
```

**Events Emitted:**

```go
// TaskCreated - New task generated
type TaskCreatedEvent struct {
    TaskID      string
    Type        TaskType
    AgentID     string
    TargetID    string
    Priority    int
    Timestamp   time.Time
}

// TaskCompleted - Task finished (success or failure)
type TaskCompletedEvent struct {
    TaskID      string
    Status      TaskStatus
    Duration    time.Duration
    SnapshotID  string // If backup task
    ErrorMsg    string // If failed
    Timestamp   time.Time
}
```

---

#### 1.5 Policy Engine Component (Future)

**Responsibilities:**
- Define backup schedules (cron-based)
- Define retention policies (keep_daily, keep_weekly, etc.)
- Evaluate when next backup should run
- Automatically create tasks based on schedule
- Check policy compliance (backups meeting SLA)
- Calculate retention windows
- Support policy inheritance (global → target-specific)
- Emit policy violation alerts

**Dependencies:**
- **External:**
  - Cron expression parser (robfig/cron)
- **Internal:**
  - Database (Policy entity)
  - TaskService (create scheduled tasks)
  - EventBus (PolicyViolation events)
  - Logger

**API Surface:**

```go
type PolicyEngine struct {
    db          *gorm.DB
    logger      core.Logger
    eventBus    core.EventBus
    taskService *TaskService
    scheduler   *cron.Cron
}

// Initialize starts policy evaluation loop
func (e *PolicyEngine) Initialize(ctx context.Context) error

// CreatePolicy defines new backup policy
func (e *PolicyEngine) CreatePolicy(ctx context.Context, policy Policy) error

// UpdatePolicy modifies existing policy
func (e *PolicyEngine) UpdatePolicy(ctx context.Context, policyID string, updates Policy) error

// DeletePolicy removes policy (fails if targets using it)
func (e *PolicyEngine) DeletePolicy(ctx context.Context, policyID string) error

// GetPolicy retrieves policy by ID
func (e *PolicyEngine) GetPolicy(ctx context.Context, policyID string) (*Policy, error)

// ListPolicies returns all policies for tenant
func (e *PolicyEngine) ListPolicies(ctx context.Context, tenantID string) ([]Policy, error)

// EvaluateCompliance checks if target meets policy
func (e *PolicyEngine) EvaluateCompliance(ctx context.Context, targetID string) (*ComplianceReport, error)

// GetNextBackupTime calculates when next backup should run
func (e *PolicyEngine) GetNextBackupTime(ctx context.Context, policyID string) (time.Time, error)

// ApplyRetention prunes snapshots per policy
func (e *PolicyEngine) ApplyRetention(ctx context.Context, targetID string) error

// ScheduleTask creates task from policy schedule (called by cron)
func (e *PolicyEngine) ScheduleTask(ctx context.Context, policyID string) error
```

**Policy Structure:**

```go
type Policy struct {
    ID          string
    TenantID    string
    Name        string
    Enabled     bool
    
    // Schedule
    Schedule    string // Cron expression (e.g., "0 2 * * *" for 2am daily)
    Timezone    string // Timezone for schedule (default: UTC)
    
    // Backup settings
    Sources     []string          // Paths to backup
    Exclude     []string          // Exclusion patterns
    Tags        []string          // Tags to apply
    
    // Retention
    Retention   RetentionPolicy
    
    // Task settings
    Priority    int               // Task priority (0-10)
    Timeout     time.Duration     // Max task duration
    
    // Compliance
    MaxAge      time.Duration     // Max snapshot age before alert
    MinSize     int64             // Min backup size (bytes)
    
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type RetentionPolicy struct {
    KeepLast    int // Keep N most recent
    KeepHourly  int // Keep hourly for N hours
    KeepDaily   int // Keep daily for N days
    KeepWeekly  int // Keep weekly for N weeks
    KeepMonthly int // Keep monthly for N months
    KeepYearly  int // Keep yearly for N years
}
```

**Compliance Evaluation:**

```go
type ComplianceReport struct {
    TargetID        string
    PolicyID        string
    Compliant       bool
    Violations      []PolicyViolation
    LastBackup      time.Time
    SnapshotAge     time.Duration
    NextBackup      time.Time
    RetentionStatus string
    EvaluatedAt     time.Time
}

type PolicyViolation struct {
    Type        ViolationType
    Severity    Severity
    Description string
    Timestamp   time.Time
}

type ViolationType string
const (
    ViolationSnapshotTooOld   ViolationType = "snapshot_too_old"
    ViolationNoRecentBackup   ViolationType = "no_recent_backup"
    ViolationRetentionViolated ViolationType = "retention_violated"
    ViolationBackupFailed     ViolationType = "backup_failed"
)

type Severity string
const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityCritical Severity = "critical"
)
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `InvalidCronExpression` | Validation | Schedule expression malformed | Fix cron syntax |
| `PolicyInUse` | Conflict | Can't delete policy with targets | Reassign targets first |
| `RetentionConflict` | Validation | Conflicting retention rules | Adjust retention settings |
| `PolicyNotFound` | NotFound | Policy ID doesn't exist | Check policy configuration |
| `SchedulingFailed` | Internal | Cron scheduler error | Check scheduler status, restart |
| `ComplianceCheckFailed` | Internal | Can't evaluate compliance | Check database, retry |

**Configuration:**

```go
type PolicyEngineConfig struct {
    EvaluationInterval time.Duration // How often to check compliance (default: 1h)
    MaxPoliciesPerTenant int          // Limit policies (default: 50)
    EnableAutoScheduling bool         // Auto-create tasks from schedules
    DefaultRetention    RetentionPolicy // Fallback retention
}
```

**Events Emitted:**

```go
// PolicyViolationEvent - Target not meeting policy
type PolicyViolationEvent struct {
    TargetID    string
    PolicyID    string
    Violation   PolicyViolation
    Timestamp   time.Time
}

// TaskScheduledEvent - Policy created scheduled task
type TaskScheduledEvent struct {
    TaskID      string
    PolicyID    string
    TargetID    string
    ScheduledAt time.Time
    Timestamp   time.Time
}
```

**Cron Schedule Examples:**

```go
// Daily at 2am
Schedule: "0 2 * * *"

// Every 6 hours
Schedule: "0 */6 * * *"

// Weekdays at 9pm
Schedule: "0 21 * * 1-5"

// First day of month at midnight
Schedule: "0 0 1 * *"
```

---

### 2. Orchestrator Frontend

The Vue 3 frontend provides the user interface for monitoring backups, managing policies, and viewing agent status.

#### 2.1 Vue Application Component

**Responsibilities:**
- Render responsive dashboard UI
- Display backup status, snapshots, and health
- Support multiple languages (i18n: German, English)
- Real-time updates via polling or WebSocket
- Client-side routing (if multi-page)
- State management for application data
- Form validation and user input handling
- Error messaging and notifications
- Dark mode support
- Export data to CSV/JSON

**Dependencies:**
- **External:**
  - Vue 3 (reactive framework)
  - Vite (build tool)
  - Tailwind CSS v4 (styling)
  - DaisyUI v5 (UI components)
  - vue-i18n (internationalization)
  - Axios or Fetch API (HTTP requests)
- **Internal:**
  - Orchestrator API (REST endpoints)

**Component Structure:**

```
frontend/src/
├── App.vue                  # Main application component
├── main.js                  # Application entry point
├── i18n.js                  # Internationalization config
├── style.css                # Global Tailwind styles
├── components/
│   ├── BackupCard.vue       # Individual backup target card
│   ├── SnapshotList.vue     # Snapshot listing table
│   ├── AgentList.vue        # Agent status grid (future)
│   ├── PolicyEditor.vue     # Policy creation form (future)
│   ├── TaskLog.vue          # Task execution logs (future)
│   ├── HealthBadge.vue      # Status indicator
│   ├── LanguageSelector.vue # i18n switcher
│   └── ThemeToggle.vue      # Dark/light mode
├── composables/
│   ├── useBackups.js        # Backup data fetching
│   ├── useAgents.js         # Agent management (future)
│   ├── usePolicies.js       # Policy management (future)
│   └── useWebSocket.js      # Real-time updates (future)
└── assets/
    └── logo.svg             # Application logo
```

**API Integration:**

```javascript
// composables/useBackups.js
import { ref, onMounted } from 'vue'

export function useBackups() {
  const backups = ref([])
  const loading = ref(true)
  const error = ref(null)
  
  async function fetchBackups() {
    try {
      loading.value = true
      const response = await fetch('/api/v1/restic/status')
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      
      const data = await response.json()
      
      // Update in-place to preserve Vue reactivity
      backups.value.forEach((backup, index) => {
        const updated = data.find(b => b.name === backup.name)
        if (updated) {
          Object.assign(backups.value[index], updated)
        }
      })
      
      // Add new backups
      data.forEach(newBackup => {
        if (!backups.value.find(b => b.name === newBackup.name)) {
          backups.value.push(newBackup)
        }
      })
      
      // Sort by name
      backups.value.sort((a, b) => a.name.localeCompare(b.name))
      
      error.value = null
    } catch (err) {
      error.value = err.message
      console.error('Failed to fetch backups:', err)
    } finally {
      loading.value = false
    }
  }
  
  async function refreshBackup(name) {
    // Refresh single backup (future API)
    await fetch(`/api/v1/restic/status/${name}/refresh`, { method: 'POST' })
    await fetchBackups()
  }
  
  onMounted(() => {
    fetchBackups()
    // Refresh every 30 seconds
    setInterval(fetchBackups, 30000)
  })
  
  return {
    backups,
    loading,
    error,
    fetchBackups,
    refreshBackup
  }
}
```

**Error Conditions:**

| Error | Type | Description | User Action |
|-------|------|-------------|-------------|
| `NetworkError` | Connection | API unreachable | Check internet, retry |
| `Unauthorized` | Auth | Token expired/invalid | Re-login |
| `ServerError` | Backend | API returned 500 | Contact admin, check logs |
| `InvalidResponse` | Parse | JSON parse failed | Report bug, check API version |
| `RateLimited` | TooManyRequests | 429 response | Reduce refresh rate |

**Configuration:**

```javascript
// vite.config.js
export default {
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: '../public',
    emptyOutDir: true
  }
}
```

**Internationalization (i18n):**

```javascript
// i18n.js
export default {
  locale: 'en',
  fallbackLocale: 'en',
  messages: {
    en: {
      dashboard: {
        title: 'Backup Dashboard',
        loading: 'Loading backups...',
        error: 'Failed to load: {msg}',
        healthy: 'Healthy',
        unhealthy: 'Unhealthy',
        disabled: 'Disabled',
        lastBackup: 'Last backup: {time}',
        snapshotAge: '{hours}h ago'
      }
    },
    de: {
      dashboard: {
        title: 'Backup-Übersicht',
        loading: 'Lade Backups...',
        error: 'Fehler beim Laden: {msg}',
        healthy: 'Gesund',
        unhealthy: 'Fehlerhaft',
        disabled: 'Deaktiviert',
        lastBackup: 'Letztes Backup: {time}',
        snapshotAge: 'vor {hours}h'
      }
    }
  }
}
```

**Responsive Design:**

```css
/* Tailwind responsive breakpoints */
/* Mobile: < 640px */
/* Tablet: 640px - 1024px */
/* Desktop: > 1024px */

.backup-grid {
  @apply grid gap-4;
  @apply grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4;
}

.backup-card {
  @apply card bg-base-100 shadow-xl;
  @apply hover:shadow-2xl transition-shadow;
}
```

---

### 3. Agent Service

The agent is a lightweight daemon running on machines to be backed up, executing Restic operations and reporting to the orchestrator.

#### 3.1 Task Polling Loop Component

**Responsibilities:**
- Poll orchestrator for pending tasks (long-polling)
- Maintain persistent connection with retry logic
- Handle network interruptions gracefully
- Queue tasks locally if offline
- Respect max concurrent task limit
- Authenticate with orchestrator (token-based)
- Handle token rotation
- Graceful shutdown (finish running tasks)

**Dependencies:**
- **External:**
  - HTTP client (net/http)
  - TLS support (crypto/tls)
- **Internal:**
  - OrchestratorClient (API communication)
  - TaskExecutor (execute tasks)
  - Config (agent configuration)
  - Logger

**API Surface:**

```go
type TaskPoller struct {
    client        *OrchestratorClient
    executor      *TaskExecutor
    pollInterval  time.Duration
    longPollTimeout time.Duration
    maxConcurrent int
    runningTasks  sync.Map // taskID -> *Task
    stopChan      chan struct{}
    wg            sync.WaitGroup
}

// Start begins polling loop
func (p *TaskPoller) Start(ctx context.Context) error

// Stop gracefully shuts down (waits for tasks to complete)
func (p *TaskPoller) Stop(ctx context.Context) error

// pollOnce performs single poll request
func (p *TaskPoller) pollOnce(ctx context.Context) ([]Task, error)

// processTasks handles retrieved tasks
func (p *TaskPoller) processTasks(ctx context.Context, tasks []Task) error

// canAcceptTask checks if under concurrency limit
func (p *TaskPoller) canAcceptTask() bool
```

**Polling Flow:**

```
┌─────────────────────────────────────┐
│ Agent Startup                       │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Authenticate with Orchestrator      │
│ (Load token from secure storage)    │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Send Initial Heartbeat              │
└──────────────┬──────────────────────┘
               │
               v
       ┌───────┴────────┐
       │ Polling Loop   │<──────┐
       └───────┬────────┘       │
               │                │
               v                │
┌─────────────────────────────────────┐
│ GET /agents/:id/tasks               │
│ (Long-poll timeout: 30s)            │
└──────────────┬──────────────────────┘
               │
         ┌─────┴─────┐
         │           │
    No tasks    Tasks found
         │           │
         │           v
         │  ┌────────────────────┐
         │  │ Execute Tasks      │
         │  │ (Up to maxConcurrent)
         │  └────────┬───────────┘
         │           │
         └───────────┴────────────┘
                     │
              Wait interval
                     │
                     └────────────────┘
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `ConnectionFailed` | Network | Can't reach orchestrator | Exponential backoff, retry |
| `Unauthorized` | Auth | Token invalid | Attempt rotation, re-register if failed |
| `OrchestratorDown` | Unavailable | 503 response | Wait, retry with backoff |
| `TaskRetrievalFailed` | Transient | Poll request failed | Log error, retry next interval |
| `ConcurrencyLimitReached` | RateLimit | Max tasks running | Wait for task completion |

**Configuration:**

```go
type PollerConfig struct {
    PollInterval     time.Duration // Time between polls (default: 60s)
    LongPollTimeout  time.Duration // Long-poll duration (default: 30s)
    MaxConcurrent    int           // Max parallel tasks (default: 2)
    RetryAttempts    int           // Connection retries (default: 5)
    RetryBackoff     time.Duration // Initial retry delay (default: 5s)
    RetryMaxBackoff  time.Duration // Max retry delay (default: 5m)
}
```

**Retry Strategy:**

```go
func (p *TaskPoller) pollWithRetry(ctx context.Context) ([]Task, error) {
    backoff := p.config.RetryBackoff
    
    for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
        tasks, err := p.pollOnce(ctx)
        if err == nil {
            return tasks, nil
        }
        
        // Log retry
        p.logger.Warn("Poll failed, retrying",
            "attempt", attempt+1,
            "backoff", backoff,
            "error", err)
        
        // Wait with exponential backoff
        select {
        case <-time.After(backoff):
            backoff *= 2
            if backoff > p.config.RetryMaxBackoff {
                backoff = p.config.RetryMaxBackoff
            }
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return nil, fmt.Errorf("max retry attempts exceeded")
}
```

---

#### 3.2 Task Executor Component

**Responsibilities:**
- Execute Restic backup/check/prune commands
- Run pre-backup plugins (database dumps, etc.)
- Stream command output to orchestrator in real-time
- Report task progress (0-100%)
- Handle command timeouts
- Capture and parse Restic JSON output
- Clean up temporary files
- Handle task cancellation (SIGTERM to Restic)
- Validate task parameters before execution

**Dependencies:**
- **External:**
  - Restic binary (os/exec)
  - Plugin binaries (mysqldump, pg_dump, etc.)
- **Internal:**
  - OrchestratorClient (status updates, log streaming)
  - Config (repository configuration)
  - Logger

**API Surface:**

```go
type TaskExecutor struct {
    client      *OrchestratorClient
    logger      core.Logger
    resticPath  string
    cacheDir    string
    configDir   string
    plugins     map[string]Plugin
}

// ExecuteTask runs task and reports progress
func (e *TaskExecutor) ExecuteTask(ctx context.Context, task Task) error

// executeBackup performs backup task
func (e *TaskExecutor) executeBackup(ctx context.Context, task Task) error

// executeCheck performs repository check
func (e *TaskExecutor) executeCheck(ctx context.Context, task Task) error

// executePrune performs snapshot pruning
func (e *TaskExecutor) executePrune(ctx context.Context, task Task) error

// runPlugin executes pre/post-backup plugin
func (e *TaskExecutor) runPlugin(ctx context.Context, plugin Plugin) error

// streamOutput sends command output to orchestrator
func (e *TaskExecutor) streamOutput(taskID string, reader io.Reader) error

// updateProgress sends progress update (0-100%)
func (e *TaskExecutor) updateProgress(taskID string, progress int) error
```

**Backup Execution Flow:**

```
┌─────────────────────────────────────┐
│ Receive Backup Task                 │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Validate Parameters                 │
│ - Check sources exist               │
│ - Validate repository config        │
│ - Check disk space                  │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Update Status: Running              │
│ POST /tasks/:id/status              │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Run Pre-Backup Plugins              │
│ - Database dumps                    │
│ - Application quiesce               │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Execute Restic Backup               │
│ restic backup /path1 /path2         │
│ --exclude=*.log --tag=daily         │
└──────────────┬──────────────────────┘
               │
      ┌────────┴────────┐
      │                 │
   Success          Failed
      │                 │
      v                 v
┌─────────────┐  ┌──────────────┐
│ Parse JSON  │  │ Capture Error│
│ Get Snap ID │  │ Message      │
└──────┬──────┘  └──────┬───────┘
       │                │
       │                │
       v                v
┌─────────────────────────────────────┐
│ Run Post-Backup Cleanup             │
│ - Remove temp files                 │
│ - Restore application state         │
└──────────────┬──────────────────────┘
               │
               v
┌─────────────────────────────────────┐
│ Update Status: Success/Failed       │
│ POST /tasks/:id/status              │
│ {snapshot_id, duration, error}      │
└─────────────────────────────────────┘
```

**Restic Command Examples:**

```go
// Backup
cmd := exec.CommandContext(ctx, e.resticPath,
    "-r", repository.URL,
    "backup",
    "/home", "/etc",
    "--exclude=*.log",
    "--exclude=.cache",
    "--tag=auto",
    "--tag=daily",
    "--json",
)

// Check
cmd := exec.CommandContext(ctx, e.resticPath,
    "-r", repository.URL,
    "check",
    "--read-data",
    "--json",
)

// Prune
cmd := exec.CommandContext(ctx, e.resticPath,
    "-r", repository.URL,
    "forget",
    "--keep-daily=7",
    "--keep-weekly=4",
    "--keep-monthly=12",
    "--prune",
    "--json",
)
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `ResticNotFound` | Fatal | Restic binary missing | Install Restic on agent |
| `SourceNotFound` | Validation | Backup path doesn't exist | Fix task parameters |
| `RepositoryLocked` | Transient | Lock file exists | Wait for lock release |
| `OutOfDiskSpace` | Resource | No space for temp files | Free disk space |
| `CommandTimeout` | Timeout | Task exceeded max duration | Increase timeout, check performance |
| `PluginFailed` | Plugin | Pre-backup plugin error | Fix plugin configuration |
| `NetworkError` | Transient | Can't reach repository | Check network, retry |
| `PermissionDenied` | Fatal | Can't read backup sources | Fix file permissions |

**Configuration:**

```go
type ExecutorConfig struct {
    ResticPath      string        // Path to restic binary
    CacheDir        string        // Restic cache directory
    ConfigDir       string        // Agent config directory
    TempDir         string        // Temporary file storage
    MaxDiskUsage    int64         // Max temp disk usage (bytes)
    CommandTimeout  time.Duration // Max command duration (default: 6h)
    StreamInterval  time.Duration // Log streaming interval (default: 5s)
}
```

**Log Streaming:**

```go
func (e *TaskExecutor) streamOutput(taskID string, reader io.Reader) error {
    scanner := bufio.NewScanner(reader)
    buffer := []LogEntry{}
    lastSend := time.Now()
    
    for scanner.Scan() {
        line := scanner.Text()
        
        buffer = append(buffer, LogEntry{
            Timestamp: time.Now(),
            Level:     "info",
            Message:   line,
        })
        
        // Send every 5 seconds or 100 lines
        if time.Since(lastSend) > e.config.StreamInterval || len(buffer) >= 100 {
            if err := e.client.AppendLogs(taskID, buffer); err != nil {
                e.logger.Warn("Failed to stream logs", "error", err)
            }
            buffer = []LogEntry{}
            lastSend = time.Now()
        }
    }
    
    // Send remaining logs
    if len(buffer) > 0 {
        return e.client.AppendLogs(taskID, buffer)
    }
    
    return scanner.Err()
}
```

---

#### 3.3 Heartbeat Monitor Component

**Responsibilities:**
- Send periodic heartbeat to orchestrator
- Report agent health metrics (CPU, memory, disk)
- Detect local system issues
- Monitor Restic process health
- Report agent version and capabilities
- Handle heartbeat failures gracefully
- Auto-reconnect after network outages

**Dependencies:**
- **External:**
  - System metrics library (shirou/gopsutil)
- **Internal:**
  - OrchestratorClient
  - Config
  - Logger

**API Surface:**

```go
type HeartbeatMonitor struct {
    client    *OrchestratorClient
    logger    core.Logger
    interval  time.Duration
    stopChan  chan struct{}
    wg        sync.WaitGroup
}

// Start begins heartbeat loop
func (m *HeartbeatMonitor) Start(ctx context.Context) error

// Stop gracefully stops heartbeat
func (m *HeartbeatMonitor) Stop(ctx context.Context) error

// collectMetrics gathers system health data
func (m *HeartbeatMonitor) collectMetrics() (*HealthMetrics, error)

// sendHeartbeat sends health data to orchestrator
func (m *HeartbeatMonitor) sendHeartbeat(ctx context.Context) error
```

**Health Metrics:**

```go
type HealthMetrics struct {
    Timestamp       time.Time
    AgentVersion    string
    Hostname        string
    
    // System metrics
    CPUPercent      float64
    MemoryUsedMB    uint64
    MemoryTotalMB   uint64
    DiskUsedGB      uint64
    DiskTotalGB     uint64
    
    // Agent metrics
    RunningTasks    int
    QueuedTasks     int
    LastBackup      time.Time
    
    // Network
    Latency         time.Duration
    
    // Capabilities
    ResticVersion   string
    AvailablePlugins []string
}
```

**Heartbeat Flow:**

```
┌─────────────────────┐
│ Every 30s           │
└──────────┬──────────┘
           │
           v
┌──────────────────────────┐
│ Collect System Metrics   │
│ - CPU, Memory, Disk      │
│ - Running tasks count    │
└──────────┬───────────────┘
           │
           v
┌──────────────────────────┐
│ POST /agents/:id/heartbeat
│ {metrics...}             │
└──────────┬───────────────┘
           │
     ┌─────┴─────┐
     │           │
 Success      Failed
     │           │
     v           v
  Continue   ┌──────────────┐
     │       │ Retry 3x     │
     │       │ Exponential  │
     │       │ Backoff      │
     │       └──────┬───────┘
     │              │
     │         ┌────┴────┐
     │         │         │
     │      Success   Failed
     │         │         │
     └─────────┴─────────┘
                         │
                         v
                  ┌──────────────┐
                  │ Log Warning  │
                  │ Continue     │
                  └──────────────┘
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `MetricsCollectionFailed` | Internal | Can't read system metrics | Log warning, send partial data |
| `HeartbeatFailed` | Network | POST request failed | Retry with backoff, continue |
| `OrchestratorUnreachable` | Network | Persistent connection failure | Queue heartbeats locally |

**Configuration:**

```go
type HeartbeatConfig struct {
    Interval        time.Duration // Heartbeat frequency (default: 30s)
    Timeout         time.Duration // Request timeout (default: 10s)
    RetryAttempts   int           // Retry count (default: 3)
    FailureThreshold int          // Consecutive failures before alert (default: 5)
}
```

---

### 4. Repository Layer

The repository layer abstracts Restic's storage backends, handling authentication and connectivity.

#### 4.1 Restic Backend Abstraction

**Responsibilities:**
- Provide unified interface for all Restic backends
- Handle backend-specific authentication
- Manage connection pooling (where applicable)
- Configure environment variables for backends
- Validate repository accessibility
- Support backend health checks
- Handle TLS certificates for HTTPS backends

**Dependencies:**
- **External:**
  - Restic binary
  - Backend services (S3 API, SFTP server, etc.)
- **Internal:**
  - Config (repository configuration)
  - Logger

**API Surface:**

```go
type Repository interface {
    // Type returns backend type (s3, sftp, local, rest, etc.)
    Type() BackendType
    
    // URL returns full Restic repository URL
    URL() string
    
    // Configure sets up environment variables and auth
    Configure(env map[string]string) error
    
    // Validate checks repository is accessible
    Validate(ctx context.Context) error
    
    // HealthCheck performs connectivity test
    HealthCheck(ctx context.Context) (bool, error)
}

type BackendType string
const (
    BackendLocal BackendType = "local"
    BackendSFTP  BackendType = "sftp"
    BackendS3    BackendType = "s3"
    BackendREST  BackendType = "rest"
    BackendAzure BackendType = "azure"
    BackendGCS   BackendType = "gcs"
)
```

**Backend Implementations:**

##### Local Filesystem Backend

```go
type LocalRepository struct {
    path         string
    passwordFile string
}

func (r *LocalRepository) Type() BackendType {
    return BackendLocal
}

func (r *LocalRepository) URL() string {
    return r.path
}

func (r *LocalRepository) Configure(env map[string]string) error {
    env["RESTIC_PASSWORD_FILE"] = r.passwordFile
    return nil
}

func (r *LocalRepository) Validate(ctx context.Context) error {
    // Check path exists and is writable
    info, err := os.Stat(r.path)
    if err != nil {
        return fmt.Errorf("repository path not accessible: %w", err)
    }
    if !info.IsDir() {
        return fmt.Errorf("repository path is not a directory")
    }
    return nil
}
```

##### S3-Compatible Backend

```go
type S3Repository struct {
    endpoint        string
    bucket          string
    region          string
    accessKeyID     string
    secretAccessKey string
    passwordFile    string
}

func (r *S3Repository) Type() BackendType {
    return BackendS3
}

func (r *S3Repository) URL() string {
    return fmt.Sprintf("s3:%s/%s", r.endpoint, r.bucket)
}

func (r *S3Repository) Configure(env map[string]string) error {
    env["RESTIC_PASSWORD_FILE"] = r.passwordFile
    env["AWS_ACCESS_KEY_ID"] = r.accessKeyID
    env["AWS_SECRET_ACCESS_KEY"] = r.secretAccessKey
    env["AWS_DEFAULT_REGION"] = r.region
    return nil
}

func (r *S3Repository) Validate(ctx context.Context) error {
    // Attempt to list objects in bucket
    cmd := exec.CommandContext(ctx, "restic", "-r", r.URL(), "snapshots", "--json")
    
    env := os.Environ()
    envMap := make(map[string]string)
    r.Configure(envMap)
    for k, v := range envMap {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    cmd.Env = env
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("repository validation failed: %w\nOutput: %s", err, output)
    }
    
    return nil
}
```

##### SFTP Backend

```go
type SFTPRepository struct {
    host         string
    port         int
    user         string
    path         string
    keyFile      string
    passwordFile string
}

func (r *SFTPRepository) Type() BackendType {
    return BackendSFTP
}

func (r *SFTPRepository) URL() string {
    return fmt.Sprintf("sftp:%s@%s:%d%s", r.user, r.host, r.port, r.path)
}

func (r *SFTPRepository) Configure(env map[string]string) error {
    env["RESTIC_PASSWORD_FILE"] = r.passwordFile
    if r.keyFile != "" {
        env["SFTP_IDENTITY_FILE"] = r.keyFile
    }
    return nil
}
```

##### REST Server Backend

```go
type RESTRepository struct {
    url          string
    username     string
    passwordFile string
    certFile     string
}

func (r *RESTRepository) Type() BackendType {
    return BackendREST
}

func (r *RESTRepository) URL() string {
    if r.username != "" {
        return fmt.Sprintf("rest:%s@%s", r.username, r.url)
    }
    return fmt.Sprintf("rest:%s", r.url)
}

func (r *RESTRepository) Configure(env map[string]string) error {
    env["RESTIC_PASSWORD_FILE"] = r.passwordFile
    if r.certFile != "" {
        env["RESTIC_CACERT"] = r.certFile
    }
    return nil
}
```

**Error Conditions:**

| Error | Type | Description | Recovery |
|-------|------|-------------|----------|
| `InvalidCredentials` | Auth | Access denied | Update credentials |
| `BucketNotFound` | Config | S3 bucket doesn't exist | Create bucket or fix config |
| `SSHKeyInvalid` | Auth | SFTP key rejected | Update SSH key |
| `CertificateExpired` | TLS | TLS cert expired | Renew certificate |
| `NetworkUnreachable` | Network | Can't connect to backend | Check network, firewall |
| `PermissionDenied` | Auth | Insufficient backend permissions | Grant access |
| `QuotaExceeded` | Resource | Storage quota exceeded | Increase quota, prune old data |

**Configuration:**

```yaml
# Example: S3 Backend
repository:
  type: s3
  endpoint: s3.amazonaws.com
  bucket: my-backups
  region: us-east-1
  access_key_id_file: /etc/restic/aws-key-id
  secret_access_key_file: /etc/restic/aws-secret
  password_file: /etc/restic/repo-password

# Example: SFTP Backend
repository:
  type: sftp
  host: backup-server.example.com
  port: 22
  user: restic
  path: /backups/server-01
  key_file: /etc/restic/ssh-key
  password_file: /etc/restic/repo-password

# Example: Local Backend
repository:
  type: local
  path: /mnt/backups
  password_file: /etc/restic/repo-password

# Example: REST Server
repository:
  type: rest
  url: https://restic.example.com/repo
  username: backup-user
  password_file: /etc/restic/rest-password
  cert_file: /etc/restic/server-ca.crt
```

---

## Summary

This comprehensive component architecture provides:

1. **Orchestrator Backend** (5 components)
   - API Server: HTTP routing, authentication, Swagger
   - Monitor Service: Direct Restic polling (current)
   - Agent Manager: Agent lifecycle (future)
   - Task Service: Task distribution (future)
   - Policy Engine: Scheduling & retention (future)

2. **Orchestrator Frontend**
   - Vue 3 SPA with i18n, real-time updates, responsive design

3. **Agent Service** (3 components)
   - Task Polling Loop: Pull-based task retrieval
   - Task Executor: Restic command execution
   - Heartbeat Monitor: Health reporting

4. **Repository Layer**
   - Backend abstraction for Local, S3, SFTP, REST, Azure, GCS
   - Unified interface with error handling

Each component includes:
- ✅ Clear responsibilities
- ✅ Dependencies and interfaces
- ✅ API surface with Go code examples
- ✅ Error conditions with recovery strategies
- ✅ Configuration options
- ✅ Implementation details

This modular architecture enables independent development, testing, and deployment of each component while maintaining clear contracts between them.

---

## Appendix

### Related Documentation

- **Agent Planning:** [agent/README.md](../agent/README.md)
- **API Documentation:** [api/swagger.yaml](../api/swagger.yaml)
- **Main README:** [README.md](../README.md)
- **AE Backend Architecture:** [AE Backend Documentation](https://github.com/AgileExecutives/ae-backend/tree/main/documentation/Architecture.md)

### Glossary

- **Orchestrator**: Central management system (AE Backend module)
- **Agent**: Lightweight daemon on backup target machines
- **Target**: A backup source (legacy direct-poll or agent-managed)
- **Repository**: Restic storage backend (S3, SFTP, local, etc.)
- **Snapshot**: Point-in-time backup created by Restic
- **Task**: Backup/check/prune operation assigned to agent
- **Policy**: Backup schedule and retention rules
- **Pull-Based**: Agents initiate connections (vs push-based)
- **Multi-Tenant**: Multiple isolated customers on shared infrastructure

### Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1 | 2025-11-23 | Initial architecture documentation |

---

**Last Updated**: November 23, 2025  
**Status**: 📋 Planning Document  
**Authors**: Alex Roehm, GitHub Copilot
