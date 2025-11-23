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
├── cmd/restic-monitor/  ← Main application entry point
├── internal/            ← Core business logic
│   ├── api/            ← REST API handlers
│   ├── config/         ← Configuration management
│   ├── monitor/        ← Restic monitoring logic
│   └── store/          ← Database models & persistence
├── frontend/           ← Vue 3 SPA
│   ├── src/
│   │   ├── App.vue    ← Main component
│   │   ├── i18n.js    ← Internationalization
│   │   └── style.css  ← Tailwind CSS v4
│   └── public/        ← Static assets
├── config/             ← Target configuration
│   └── targets.json   ← Repository definitions
├── api/                ← OpenAPI/Swagger documentation
├── data/               ← SQLite database
└── agent/              ← Future: Backup Agent (coming soon)

---

## 🧭 Feature Roadmap

### Phase 1 — Foundations ✅

✅ Architectural plan  
✅ Multi-target monitoring  
✅ Health checks with customizable age validation  
✅ Prune with retention policies  
✅ Swagger API documentation  
✅ Mock mode for development  

### Phase 2 — Backup Agent (Coming Soon)

⬜ Go agent binary for Linux/Windows/macOS  
⬜ Agent installation scripts  
⬜ Task execution engine (restic backup/check/prune)  
⬜ Secure token storage  
⬜ API for agent registration & heartbeat  

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

* No agent system yet (manual Restic setup required)
* No task scheduling (monitoring only)
* No backup automation
* No centralized policy management

These limitations will be removed as the orchestrator/agent architecture is implemented.

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

Contributions are welcome! The upcoming agent architecture is documented in the roadmap above.

**Areas for contribution:**
- Agent system design and implementation
- Additional UI features
- Tests and documentation
- Internationalization (new languages)
- Plugin system for custom backup sources

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

- [Swagger API Documentation](http://localhost:8080/api/v1/swagger) (when `SHOW_SWAGGER=true`)
- [Makefile Commands](Makefile) - See `make help` for all available commands
- [Example Configuration](config/targets.example.json)

---

**Built with ❤️ for the Restic community**
