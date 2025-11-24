# 📦 restic-monitor

**Centralized Restic Backup Monitoring & (future) Management Platform**

`restic-monitor` is a Go-based backend with a Vue frontend designed to monitor the health, status, and snapshots of Restic backup repositories. It provides an easy way to visualize backup status across multiple Restic targets such as S3, MinIO, rest-server, and local filesystems.

This project currently focuses on **monitoring**, but is being expanded into a full **centralized backup management system** with lightweight cross-platform **backup agents**.

---

## 🚀 Current Features (v0.x)

- 📊 **Beautiful Dashboard** - Modern UI built with Vue 3, Tailwind CSS v4, and DaisyUI v5
- 🔍 **Multi-Target Monitoring** - Monitor multiple Restic repositories simultaneously
- ✅ **Health Checks** - Automatic repository health validation with customizable age checks
- 🔓 **Repository Unlock** - Unlock locked repositories with one click
- 🗑️ **Prune Operations** - Configurable retention policies (keep-last, keep-daily, keep-weekly, keep-monthly)
- 📸 **Snapshot Browser** - View all snapshots with metadata and file lists
- 🌍 **Internationalization** - Built-in English and German translations
- 🎨 **Dark Mode** - Light and dark theme support with smooth transitions
- 📱 **Responsive Design** - Works on desktop, tablet, and mobile
- 🔄 **Auto-Refresh** - Real-time status updates with smooth animations
- 🔐 **Authentication** - Basic Auth and Bearer Token support
- 📚 **API Documentation** - Interactive Swagger UI (optional)
- 🐳 **Docker Support** - Easy deployment with Docker and Docker Compose
- 🧪 **Mock Mode** - Development mode for testing without real Restic repositories

---

## 🛠️ Work in Progress — New Architecture Roadmap

The project is evolving from a simple monitoring dashboard into a **complete backup orchestration platform**.

### 🏗️ Planned Architecture

`restic-monitor` will become the **central orchestrator**, responsible for:

* Managing backup policies
* Distributing backup tasks
* Tracking agent health
* Aggregating logs
* Coordinating pruning and verification
* Providing a modern UI for backup operations

A new cross-platform **Backup Agent** (written in Go) will run on every machine that needs to be backed up.

### 📡 Pull-Based Communication Model

Each agent will:

* Register itself with the orchestrator
* Send periodic heartbeats
* Poll for tasks (backup / check / prune)
* Execute Restic commands locally
* Send logs and status updates back to the orchestrator

This design avoids firewall issues and works across Linux, Windows, and macOS.

---

## 🧩 Repository Structure (in Transition)

```
restic-monitor/
│
├── cmd/                     ← Current: Orchestrator entry point
│   └── restic-monitor/
│       └── main.go
│
├── internal/                ← Current: Orchestrator business logic
│   ├── api/                ← REST API handlers
│   ├── config/             ← Configuration management
│   ├── monitor/            ← Restic monitoring logic
│   └── store/              ← Database models & persistence
│
├── frontend/                ← Current: Vue 3 web interface
│   ├── src/
│   │   ├── App.vue        ← Main component
│   │   ├── i18n.js        ← Internationalization
│   │   └── style.css      ← Tailwind CSS v4
│   └── public/            ← Static assets
│
├── api/                     ← Current: OpenAPI documentation
│   ├── swagger.yaml
│   ├── swagger.json
│   └── docs.go
│
├── config/                  ← Current: Target configuration
│   ├── targets.json        ← Repository definitions
│   └── targets.example.json
│
├── data/                    ← Current: SQLite database
├── public/                  ← Current: Snapshot file lists
│
└── agent/                   ← Future: Backup Agent (not yet implemented)
    └── README.md           ← Agent planning documentation
```

**Current Structure:** All existing code represents the **orchestrator** (central monitoring system)

**Future Structure:** The `agent/` directory contains planning documentation for the distributed backup agent that will be developed in 2026.

### Orchestrator vs Agent

| Component | Location | Status | Purpose |
|-----------|----------|--------|---------|
| **Orchestrator** | `cmd/`, `internal/`, `frontend/`, `api/` | ✅ Active | Central monitoring & management system |
| **Agent** | `agent/` | 📋 Planned | Lightweight backup agent for target machines |

The current codebase is the orchestrator. No restructuring of existing files is needed at this time.

---

## 🏗️ Architecture & Design

### Current: Centralized Monitoring (v0.x)

The current implementation is a **centralized orchestrator** that:
- Polls Restic repositories directly
- Requires network access to all backup targets
- Stores status and snapshots in SQLite
- Provides a web UI for visualization

```
┌─────────────────────────────────────┐
│   Restic Monitor (Orchestrator)     │
│  ┌──────────────┐  ┌──────────────┐ │
│  │   Vue UI     │  │   Go API     │ │
│  └──────────────┘  └──────────────┘ │
│         │                 │          │
│         └────────┬────────┘          │
│                  │                   │
│           ┌──────▼──────┐           │
│           │   SQLite    │           │
│           └─────────────┘           │
└────────────────┬────────────────────┘
                 │
      ┌──────────┼──────────┐
      │          │          │
┌─────▼───┐ ┌───▼────┐ ┌───▼────┐
│ Restic  │ │ Restic │ │ Restic │
│ Repo 1  │ │ Repo 2 │ │ Repo 3 │
└─────────┘ └────────┘ └────────┘
```

### Future: Distributed Agent System (v1.x+)

The planned architecture introduces **lightweight agents** on each backup target:

```
┌─────────────────────────────────────────┐
│    Restic Monitor (Orchestrator)        │
│  ┌──────────────┐  ┌──────────────┐    │
│  │   Vue UI     │  │   Go API     │    │
│  │              │  │              │    │
│  │ - Dashboard  │  │ - Policies   │    │
│  │ - Logs       │  │ - Scheduling │    │
│  │ - Policies   │  │ - Agent Mgmt │    │
│  └──────────────┘  └──────────────┘    │
│         │                 │             │
│         └────────┬────────┘             │
│                  │                      │
│           ┌──────▼──────┐              │
│           │  Database   │              │
│           │  (Postgres) │              │
│           └─────────────┘              │
└───────────────┬────────────────────────┘
                │ HTTPS (Pull-based)
                │
    ┌───────────┼───────────┐
    │           │           │
┌───▼────┐  ┌──▼─────┐  ┌──▼─────┐
│ Agent  │  │ Agent  │  │ Agent  │
│        │  │        │  │        │
│ ┌────┐ │  │ ┌────┐ │  │ ┌────┐ │
│ │Res │ │  │ │Res │ │  │ │Res │ │
│ │tic │ │  │ │tic │ │  │ │tic │ │
│ └────┘ │  │ └────┘ │  │ └────┘ │
└────────┘  └────────┘  └────────┘
 Server 1    Server 2    Server 3
```

**Agent Responsibilities:**
- Register with orchestrator
- Poll for backup tasks
- Execute Restic locally
- Stream logs and status
- Report health metrics

**Orchestrator Responsibilities:**
- Define backup policies
- Schedule tasks
- Track agent health
- Aggregate logs and metrics
- Provide unified UI

### Why Pull-Based?

The agent uses a **pull model** instead of push:

✅ **Firewall Friendly**: Agents initiate connections (no inbound ports)  
✅ **NAT Traversal**: Works behind NAT without port forwarding  
✅ **Secure**: One-way trust (agents trust orchestrator, not vice versa)  
✅ **Scalable**: Orchestrator doesn't need to track agent IPs  
✅ **Resilient**: Agents reconnect automatically after network issues  

---

## 🧭 Feature Roadmap

### Phase 1 — Foundations ✅

✅ Architectural plan  
✅ Multi-target monitoring  
✅ Health checks with customizable age validation  
✅ Prune with retention policies  
✅ Swagger API documentation  
✅ Mock mode for development  

### Phase 2 — Backup Agent (2026)

⬜ Agent architecture design and planning  
⬜ Go agent binary for Linux/Windows/macOS  
⬜ Agent installation scripts and systemd/Windows service integration  
⬜ Task execution engine (restic backup/check/prune)  
⬜ Secure token storage and authentication  
⬜ API for agent registration & heartbeat  
⬜ Long-polling or WebSocket for task distribution  

**See:** [`agent/README.md`](agent/README.md) for detailed agent planning

### Phase 3 — UI Enhancements

⬜ Machine overview  
⬜ Policy editor  
⬜ Backup history & logs view  
⬜ Real-time log streaming  

### Phase 4 — Advanced Features

⬜ Multi-repository routing  
⬜ Auto-update system for agents  
⬜ Notifications (email/Slack)  
⬜ Plugin system for DB dumps / Docker volumes / VM snapshots  

---

## 🔍 Current Limitations

* **No agent system** - Orchestrator must have direct network access to all Restic repositories
* **No distributed architecture** - All monitoring runs centrally
* **No task scheduling** - Monitoring only (no automated backups)
* **No backup automation** - Manual Restic setup required on each machine
* **No centralized policy management** - Retention policies configured per-target

These limitations will be addressed with the agent system (see [`agent/README.md`](agent/README.md))

---

## 📥 Installation

### Quick Start with Docker Compose (Recommended)

1. Clone the repository:

```bash
git clone https://github.com/AlexRoehm/restic-monitor.git
cd restic-monitor
```

2. Create a `config/targets.json` file:

```json
[
  {
    "name": "home",
    "repository": "/path/to/home/backup",
    "password_file": "/etc/restic/home.pass",
    "keep_last": 10,
    "keep_daily": 7,
    "keep_weekly": 4,
    "keep_monthly": 6
  },
  {
    "name": "work",
    "repository": "rest:https://backup.example.com/work",
    "password": "your-restic-password",
    "certificate_file": "/etc/restic/ca.pem",
    "keep_last": 5,
    "keep_daily": 4,
    "keep_weekly": 3,
    "keep_monthly": 2
  },
  {
    "name": "archive",
    "repository": "/mnt/backup/archive",
    "password": "your-password",
    "disabled": true,
    "keep_last": 3
  }
]
```

3. Run with Docker Compose:

```bash
docker-compose up -d
```

4. Open http://localhost:8080 in your browser

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RESTIC_BINARY` | `restic` | Path to restic binary |
| `DATABASE_DSN` | `restic-monitor.db` | SQLite database path |
| `API_LISTEN_ADDR` | `:8080` | API server listen address |
| `CHECK_INTERVAL` | `10m` | Interval between backup checks |
| `RESTIC_TIMEOUT` | `3m` | Timeout for restic CLI commands |
| `SNAPSHOT_FILE_LIMIT` | `200` | Maximum number of files to list per snapshot |
| `TARGETS_FILE` | `config/targets.json` | Path to targets configuration file |
| `STATIC_DIR` | `frontend/dist` | Frontend static files directory |
| `PUBLIC_DIR` | `public` | Directory for snapshot file lists |
| `AUTH_USERNAME` | _(empty)_ | Basic auth username (optional) |
| `AUTH_PASSWORD` | _(empty)_ | Basic auth password (optional) |
| `AUTH_TOKEN` | _(empty)_ | API bearer token (optional) |
| `SHOW_SWAGGER` | `false` | Enable Swagger UI at `/api/v1/swagger` |
| `MOCK_MODE` | `false` | Mock restic calls for development |

**Note**: Authentication can be enabled using either:
- **Basic Auth**: Set both `AUTH_USERNAME` and `AUTH_PASSWORD`
- **Bearer Token**: Set `AUTH_TOKEN` for API clients
- **Both**: Both methods will be accepted if all three variables are set

The frontend will prompt for Basic Auth credentials when needed. API clients can use `Authorization: Bearer <token>` header.

### Target Configuration

The `targets.json` file configures which Restic repositories to monitor:

```json
[
  {
    "name": "my-backup",
    "repository": "rest:https://backup.example.com/repo",
    "password": "secret",
    "passwordFile": "",
    "certificateFile": "/certs/ca.pem"
  }
]
```

**Fields:**
- `name` - Unique identifier for the target
- `repository` - Restic repository URL or path
- `password` - Restic repository password (optional if using password_file)
- `password_file` - Path to file containing password (optional if using password)
- `certificate_file` - Path to CA certificate for HTTPS repositories (optional)
- `disabled` - Set to `true` to skip monitoring this target (optional)
- `keep_last` - Number of latest snapshots to keep during prune (optional)
- `keep_daily` - Number of daily snapshots to keep (optional)
- `keep_weekly` - Number of weekly snapshots to keep (optional)
- `keep_monthly` - Number of monthly snapshots to keep (optional)

### Supported Repository Types

- Local: `/path/to/repo`
- SFTP: `sftp:user@host:/path/to/repo`
- REST: `rest:https://user:pass@host:8000/`
- S3: `s3:s3.amazonaws.com/bucket/path`
- Azure: `azure:container:/path`
- B2: `b2:bucketname:/path`
- And more - see [Restic documentation](https://restic.readthedocs.io/)

---

## 🛠️ Development

### Prerequisites

- Go 1.24 or later
- Node.js 20 or later
- Restic CLI (for production mode)

### Makefile Commands

```bash
make help          # Show all available commands
make build         # Build both backend and frontend
make run           # Run the application
make dev-backend   # Run backend in development mode
make dev-frontend  # Run frontend dev server
make watch         # Watch and rebuild backend on changes (requires air)
make swagger       # Generate Swagger/OpenAPI documentation
make test          # Run Go tests
make docker-build  # Build Docker image
make clean         # Clean build artifacts
```

### Build from Source

1. Clone the repository:
```bash
git clone https://github.com/AlexRoehm/restic-monitor.git
cd restic-monitor
```

2. Install dependencies:
```bash
make install  # Installs both Go and npm dependencies
```

3. Build:
```bash
make build
```

4. Configure:
```bash
cp .env.example .env
cp config/targets.example.json config/targets.json
# Edit .env and config/targets.json
```

5. Run:
```bash
make run
```

### Development Mode with Mock Data

For development without real Restic repositories:

```bash
# In .env, set:
MOCK_MODE=true
```

This will:
- Return fake snapshot data
- Skip actual restic command execution
- Show one disabled and one unhealthy target for testing
- Allow testing all UI features without real backups

### Development Servers

Run backend and frontend separately for hot-reload:

**Terminal 1 - Backend:**
```bash
make dev-backend
# or with auto-reload:
make watch
```

**Terminal 2 - Frontend:**
```bash
make dev-frontend
```

Frontend runs on http://localhost:5173 and proxies API requests to backend on http://localhost:8080.

---

## 📡 API Documentation

### Interactive API Explorer

When `SHOW_SWAGGER=true` is set, visit:

```
http://localhost:8080/api/v1/swagger
```

### Main Endpoints

#### GET `/api/v1/status`

Get status of all targets or a specific target.

**Query Parameters:**
- `name` (optional) - Filter by target name

**Response:**
```json
[
  {
    "name": "home",
    "latestBackup": "2025-11-23T10:30:00Z",
    "latestSnapshotID": "a1b2c3d4",
    "snapshotCount": 42,
    "fileCount": 1234,
    "health": true,
    "statusMessage": "",
    "checkedAt": "2025-11-23T15:45:00Z",
    "disabled": false
  }
]
```

#### GET `/api/v1/status/{name}`

Get status of a specific target with optional age validation.

**Path Parameters:**
- `name` - Target name

**Query Parameters:**
- `maxage` (optional) - Maximum age in hours. Returns healthy only if repository is healthy AND latest snapshot is younger than maxage hours.

**Example:**
```bash
# Check if backup is healthy and less than 24 hours old
curl http://localhost:8080/api/v1/status/home?maxage=24
```

#### GET `/api/v1/snapshots/{name}`

Get all snapshots for a target.

**Response:**
```json
[
  {
    "short_id": "a1b2c3d4",
    "time": "2025-11-23T14:30:00Z",
    "hostname": "myserver",
    "username": "admin",
    "paths": ["/home/user", "/etc"],
    "tags": ["daily", "production"]
  }
]
```

#### GET `/api/v1/snapshot/{id}`

Get file list for a specific snapshot.

#### POST `/api/v1/unlock/{name}`

Unlock a locked repository.

**Response:**
```json
{
  "status": "unlocked",
  "target": "home",
  "message": "successfully removed locks"
}
```

#### POST `/api/v1/prune/{name}`

Prune snapshots according to retention policy. Use `all` to prune all targets.

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/prune/home
curl -X POST http://localhost:8080/api/v1/prune/all
```

#### POST `/api/v1/toggle/{name}`

Enable or disable monitoring for a target.

---

## 🏗️ Architecture

- **Backend**: Go 1.24 with GORM (SQLite), periodic Restic execution
- **Frontend**: Vue 3 with Composition API, Tailwind CSS v4, DaisyUI v5
- **Database**: SQLite (embedded, no external DB required)
- **API**: RESTful HTTP with OpenAPI/Swagger documentation
- **File Storage**: JSONL files for snapshot file lists
- **Internationalization**: vue-i18n (English and German)

### Technology Stack

**Backend:**
- Go 1.24
- GORM v2 (SQLite driver)
- Standard library HTTP server
- swaggo/swag for API documentation

**Frontend:**
- Vue 3 (Composition API)
- Vite 6
- Tailwind CSS v4
- DaisyUI v5
- vue-i18n v10

---

## 🔒 Security Considerations

- Store passwords securely using `password_file` instead of plain text
- Use HTTPS for remote repositories
- Validate certificate files for TLS connections
- Enable authentication in production (`AUTH_USERNAME`/`AUTH_PASSWORD` or `AUTH_TOKEN`)
- Run container as non-root user
- Mount sensitive files read-only in Docker
- Keep `targets.json` with credentials outside version control

---

## 🐳 Docker Deployment

### Docker Run

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/public:/app/public \
  -e SHOW_SWAGGER=true \
  -e AUTH_USERNAME=admin \
  -e AUTH_PASSWORD=admin \
  --name restic-monitor \
  guxxde/restic-monitor:latest
```

### Multi-Architecture Support

Docker images are built for:
- `linux/amd64`
- `linux/arm64`

---

## 🧪 Testing

```bash
make test
```

---

## 🤝 Contributing

Contributions are welcome!

**Current Focus Areas:**
- Bug fixes and improvements to orchestrator
- UI/UX enhancements
- Additional API endpoints
- Documentation improvements
- Tests and test coverage

**Future Development:**
- Agent system design (see [`agent/README.md`](agent/README.md))
- Distributed architecture planning
- Plugin system for custom backup sources

**How to Contribute:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

**Discussions:**
- Use `[orchestrator]` prefix for current system improvements
- Use `[agent]` prefix for agent-related proposals
- Use `[architecture]` prefix for system-wide design discussions

Please open an issue to discuss major changes before submitting a PR.

---

## 📜 License

MIT License - see LICENSE file for details

---

## 🆘 Support

- [GitHub Issues](https://github.com/AlexRoehm/restic-monitor/issues)
- [Restic Documentation](https://restic.readthedocs.io/)

---

## 📚 Additional Resources

- **Architecture Documentation:** [`docs/architecture.md`](docs/architecture.md) - Complete system architecture
- **Agent Planning:** [`agent/README.md`](agent/README.md) - Detailed agent system design
- **API Documentation:** [Swagger UI](http://localhost:8080/api/v1/swagger) (when `SHOW_SWAGGER=true`)
- **Makefile Commands:** See `make help` for all available commands
- **Example Configuration:** [`config/targets.example.json`](config/targets.example.json)
- **Restic Documentation:** [restic.readthedocs.io](https://restic.readthedocs.io/)
- **AE Backend:** [github.com/AgileExecutives/ae-backend](https://github.com/AgileExecutives/ae-backend)

---

**Built with ❤️ for the Restic community**
