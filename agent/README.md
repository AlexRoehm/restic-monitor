# Restic Monitor Agent

**Status:** 🚧 Planned - Not Yet Implemented

The Restic Monitor Agent is a lightweight, cross-platform backup agent that will run on machines that need to be backed up. It communicates with the central orchestrator using a pull-based model.

## Architecture Overview

### Pull-Based Communication

The agent uses a **pull-based** communication model to avoid firewall issues:

1. **Registration**: Agent registers itself with the orchestrator on first startup
2. **Heartbeat**: Sends periodic heartbeats to indicate it's alive
3. **Task Polling**: Polls the orchestrator for pending tasks (backup/check/prune)
4. **Execution**: Runs Restic commands locally
5. **Reporting**: Sends logs and status updates back to orchestrator

### Planned Features

- ✅ **Cross-Platform**: Windows, Linux, macOS support
- ✅ **Lightweight**: Minimal resource footprint
- ✅ **Secure**: Token-based authentication with orchestrator
- ✅ **Resilient**: Works behind NAT/firewalls
- ✅ **Automated**: Schedule-based and policy-driven backups
- ✅ **Extensible**: Plugin system for custom backup sources

## Directory Structure (Planned)

```
agent/
├── cmd/
│   └── agent/
│       └── main.go           # Agent entry point
├── pkg/
│   ├── client/               # Orchestrator API client
│   ├── executor/             # Restic command executor
│   ├── scheduler/            # Task scheduling
│   ├── heartbeat/            # Health monitoring
│   └── config/               # Agent configuration
├── plugins/                  # Extensible backup sources
│   ├── mysql/               # MySQL dump plugin
│   ├── postgres/            # PostgreSQL dump plugin
│   └── docker/              # Docker volume plugin
└── README.md
```

## Planned Components

### 1. Orchestrator Client (`pkg/client/`)

Handles communication with the central orchestrator:
- Registration and authentication
- Task polling (long-polling or intervals)
- Status reporting
- Log streaming

### 2. Task Executor (`pkg/executor/`)

Executes Restic operations:
- `backup` - Create new snapshots
- `check` - Verify repository integrity
- `prune` - Remove old snapshots per policy
- `forget` - Delete specific snapshots

### 3. Scheduler (`pkg/scheduler/`)

Manages task execution:
- Cron-like scheduling
- Policy-based triggers
- Resource throttling
- Retry logic

### 4. Heartbeat (`pkg/heartbeat/`)

Monitors agent health:
- Periodic status updates
- Resource usage reporting
- Network connectivity checks
- Disk space monitoring

### 5. Plugin System (`plugins/`)

Extensible backup sources:
- Database dumps (MySQL, PostgreSQL, MongoDB)
- Docker volumes
- VM snapshots
- Application-specific backups

## Configuration (Planned)

The agent will use a YAML configuration file:

```yaml
orchestrator:
  url: https://restic-monitor.example.com
  token: <agent-token>
  
agent:
  id: server-01
  hostname: web-server
  poll_interval: 60s
  heartbeat_interval: 30s

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

plugins:
  mysql:
    enabled: true
    host: localhost
    user: backup
    password_file: /etc/mysql/backup.pass
```

## Installation (Future)

### Linux (systemd)

```bash
# Download agent
wget https://github.com/AlexRoehm/restic-monitor/releases/download/v1.0.0/restic-agent-linux-amd64
chmod +x restic-agent-linux-amd64
sudo mv restic-agent-linux-amd64 /usr/local/bin/restic-agent

# Create configuration
sudo mkdir -p /etc/restic-agent
sudo nano /etc/restic-agent/config.yaml

# Install systemd service
sudo restic-agent install
sudo systemctl enable restic-agent
sudo systemctl start restic-agent
```

### Windows (Service)

```powershell
# Download agent
Invoke-WebRequest -Uri https://github.com/AlexRoehm/restic-monitor/releases/download/v1.0.0/restic-agent-windows-amd64.exe -OutFile restic-agent.exe

# Create configuration
New-Item -Path "C:\Program Files\ResticAgent" -ItemType Directory
notepad "C:\Program Files\ResticAgent\config.yaml"

# Install as Windows service
.\restic-agent.exe install
Start-Service ResticAgent
```

### macOS (launchd)

```bash
# Download agent
brew install restic-agent

# Create configuration
mkdir -p ~/.config/restic-agent
nano ~/.config/restic-agent/config.yaml

# Install launch agent
restic-agent install --user
launchctl load ~/Library/LaunchAgents/com.restic-monitor.agent.plist
```

## Development Timeline

### Phase 1: Core Agent (Q1 2026)
- [ ] Basic orchestrator client
- [ ] Task polling mechanism
- [ ] Restic executor
- [ ] Configuration management
- [ ] Logging and error handling

### Phase 2: Platform Support (Q2 2026)
- [ ] Windows service integration
- [ ] Linux systemd integration
- [ ] macOS launchd integration
- [ ] Cross-compilation builds
- [ ] Installation scripts

### Phase 3: Advanced Features (Q3 2026)
- [ ] Plugin system
- [ ] Database backup plugins
- [ ] Docker volume support
- [ ] Resource monitoring
- [ ] Auto-update mechanism

### Phase 4: Production Ready (Q4 2026)
- [ ] Comprehensive testing
- [ ] Security audit
- [ ] Performance optimization
- [ ] Documentation
- [ ] Release v1.0.0

## API Contract (Planned)

The agent will communicate with these orchestrator endpoints:

### POST `/api/v1/agent/register`
Register a new agent with the orchestrator.

### POST `/api/v1/agent/heartbeat`
Send periodic health status.

### GET `/api/v1/agent/tasks`
Poll for pending tasks (long-polling supported).

### POST `/api/v1/agent/tasks/{id}/status`
Update task execution status.

### POST `/api/v1/agent/logs`
Stream logs to orchestrator.

## Security Considerations

- **Token-Based Auth**: Each agent has a unique token
- **TLS Required**: All communication over HTTPS
- **Token Rotation**: Support for periodic token refresh
- **Least Privilege**: Agent runs with minimal permissions
- **Secure Storage**: Credentials stored in OS keyring/credential manager

## Contributing

The agent system is in the planning phase. Design discussions and contributions are welcome!

**Discussion Topics:**
- Agent-orchestrator protocol design
- Cross-platform compatibility approach
- Plugin system architecture
- Security model
- Deployment strategies

Please open issues or PRs with `[agent]` prefix for agent-related discussions.

## Related Documentation

- [Orchestrator README](../README.md) - Central monitoring system
- [Architecture Overview](../docs/architecture.md) - Full system design (coming soon)
- [API Documentation](../api/swagger.yaml) - Orchestrator API spec

---

**Note:** This component is part of the larger Restic Monitor ecosystem. The current release focuses on the orchestrator (monitoring). Agent development will begin in 2026.
