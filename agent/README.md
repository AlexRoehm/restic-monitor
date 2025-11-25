# Restic Monitor Agent

**Status:** 🔨 IN PROGRESS - Configuration System Implemented (EPIC 8.1 Complete)

The Restic Monitor Agent is a lightweight, cross-platform backup agent that runs on machines that need to be backed up. It communicates with the central orchestrator using a pull-based model.

## Current Status (EPIC 8)

✅ **User Story 8.1 COMPLETE:** Agent Configuration Format & Loader  
- 14 configuration fields with comprehensive validation
- Environment variable overrides (13 RESTIC_AGENT_* variables)
- YAML-based configuration with sensible defaults
- 14 test functions, 30 test cases passing

🔲 **Next:** User Story 8.2 - Agent Identity Persistence  
🔲 **Then:** User Story 8.3 - First-Run Registration Workflow

See `docs/epic8-status.md` for complete roadmap.

## Quick Start

### Configuration File

Create `/etc/restic-agent/agent.yaml`:

```yaml
# Required fields
orchestratorUrl: "https://backup.example.com"
authenticationToken: "your-secret-token"

# Optional fields (showing defaults)
logLevel: "info"                      # debug, info, warn, error
pollingIntervalSeconds: 30            # 5-3600 seconds
heartbeatIntervalSeconds: 60          # 10-3600 seconds
maxConcurrentJobs: 2                  # 1-10 jobs
httpTimeoutSeconds: 30                # 5-300 seconds
retryMaxAttempts: 3                   # 0-10 attempts
retryBackoffSeconds: 5                # 1-60 seconds
```

### Environment Variables

Override configuration values:

```bash
export RESTIC_AGENT_ORCHESTRATOR_URL="https://backup.example.com"
export RESTIC_AGENT_AUTH_TOKEN="secret-token"
export RESTIC_AGENT_LOG_LEVEL="debug"
export RESTIC_AGENT_POLLING_INTERVAL="15"
```

See `docs/agent/configuration-spec.md` for complete documentation.

## Configuration Management

The agent package provides robust configuration loading with:

### Features
- ✅ YAML-based configuration format
- ✅ Environment variable overrides
- ✅ Sensible defaults for optional fields
- ✅ Comprehensive validation with detailed error messages
- ✅ URL format validation
- ✅ Range validation for all numeric fields
- ✅ Log level enum validation

### Usage

```go
import "github.com/example/restic-monitor/agent"

cfg, err := agent.LoadConfig("/etc/restic-agent/agent.yaml")
if err != nil {
    log.Fatalf("Configuration error: %v", err)
}

fmt.Printf("Orchestrator: %s\n", cfg.OrchestratorURL)
fmt.Printf("Polling interval: %d seconds\n", cfg.PollingIntervalSeconds)
```

### Validation

The configuration loader validates:
- ✅ Required fields (orchestratorUrl, authenticationToken)
- ✅ URL format (http/https only, no trailing slash)
- ✅ Log level (debug, info, warn, error)
- ✅ Polling interval (5-3600 seconds)
- ✅ Heartbeat interval (10-3600 seconds)
- ✅ Max concurrent jobs (1-10)
- ✅ HTTP timeout (5-300 seconds)
- ✅ Retry settings (0-10 attempts, 1-60 second backoff)

### Error Handling

Detailed error messages help troubleshoot configuration issues:

```
configuration validation failed: orchestratorUrl is required; 
pollingIntervalSeconds must be between 5 and 3600; 
logLevel must be one of: debug, info, warn, error
```

## Testing

Comprehensive test coverage with TDD methodology:

```bash
go test ./agent/... -v
```

**Test Coverage (14 functions, 30 test cases):**
- ✅ Minimal valid configuration
- ✅ Full configuration with all fields
- ✅ Missing required fields
- ✅ Invalid YAML parsing
- ✅ File not found errors
- ✅ Range validation (pollingIntervalSeconds, maxConcurrentJobs)
- ✅ Enum validation (logLevel)
- ✅ Environment variable overrides
- ✅ URL format validation (protocol, trailing slash)
- ✅ Default value application

## Documentation

- **Configuration Spec:** `docs/agent/configuration-spec.md` (650+ lines)
- **EPIC 8 Status:** `docs/epic8-status.md`
- **Architecture:** `docs/architecture.md`

## Roadmap (EPIC 8)

## Roadmap (EPIC 8)

### ✅ Phase 1: Configuration System (COMPLETE)
- ✅ YAML configuration format
- ✅ Environment variable overrides
- ✅ Comprehensive validation
- ✅ 14 test functions, 30 test cases

### 🔲 Phase 2: Identity & Registration (IN PROGRESS)
- [ ] State file persistence (state.json)
- [ ] Agent UUID generation
- [ ] POST /agents/register API call
- [ ] First-run registration workflow

### 🔲 Phase 3: Installation & Bootstrap
- [ ] Linux installation script (systemd)
- [ ] macOS installation script (launchd)
- [ ] Windows installation script (service)
- [ ] Agent diagnostics (--test-config)
- [ ] Bootstrap logging

### 🔲 Phase 4: Core Agent Runtime
- [ ] Heartbeat mechanism
- [ ] Task polling
- [ ] Restic executor
- [ ] Job scheduling
- [ ] Log streaming

### 🔲 Phase 5: Advanced Features
- [ ] Plugin system
- [ ] Database backup plugins
- [ ] Docker volume support
- [ ] Resource monitoring
- [ ] Auto-update mechanism

## Configuration Fields

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|------------|
| orchestratorUrl | string | Yes | - | http/https, no trailing slash |
| authenticationToken | string | Yes | - | Non-empty |
| agentId | string | No | - | UUID format |
| hostnameOverride | string | No | - | - |
| logLevel | string | No | info | debug, info, warn, error |
| logFile | string | No | - | Valid path |
| pollingIntervalSeconds | int | No | 30 | 5-3600 |
| heartbeatIntervalSeconds | int | No | 60 | 10-3600 |
| maxConcurrentJobs | int | No | 2 | 1-10 |
| httpTimeoutSeconds | int | No | 30 | 5-300 |
| retryMaxAttempts | int | No | 3 | 0-10 |
| retryBackoffSeconds | int | No | 5 | 1-60 |
| stateFile | string | No | /var/lib/restic-agent/state.json | - |
| tempDir | string | No | /tmp/restic-agent | - |

## Environment Variables

All configuration fields can be overridden via environment variables:

- `RESTIC_AGENT_ORCHESTRATOR_URL`
- `RESTIC_AGENT_AUTH_TOKEN`
- `RESTIC_AGENT_ID`
- `RESTIC_AGENT_HOSTNAME`
- `RESTIC_AGENT_LOG_LEVEL`
- `RESTIC_AGENT_LOG_FILE`
- `RESTIC_AGENT_POLLING_INTERVAL`
- `RESTIC_AGENT_HEARTBEAT_INTERVAL`
- `RESTIC_AGENT_MAX_CONCURRENT_JOBS`
- `RESTIC_AGENT_HTTP_TIMEOUT`
- `RESTIC_AGENT_STATE_FILE`
- `RESTIC_AGENT_TEMP_DIR`

## Security Considerations

1. **Token Storage:** Never commit `authenticationToken` to version control
2. **Environment Variables:** Use `RESTIC_AGENT_AUTH_TOKEN` for sensitive values
3. **File Permissions:** Set `agent.yaml` to `0600` (owner read/write only)
4. **URL Validation:** Only `http`/`https` protocols allowed
5. **State File:** Protect `state.json` with appropriate permissions

## Examples

### Minimal Configuration

```yaml
orchestratorUrl: "https://backup.example.com"
authenticationToken: "abc123..."
```

### Production Configuration

```yaml
orchestratorUrl: "https://backup.example.com:8443"
authenticationToken: "${RESTIC_AGENT_AUTH_TOKEN}"
logLevel: "info"
logFile: "/var/log/restic-agent/agent.log"
pollingIntervalSeconds: 15
heartbeatIntervalSeconds: 30
maxConcurrentJobs: 4
httpTimeoutSeconds: 60
retryMaxAttempts: 5
retryBackoffSeconds: 10
```

### Development Configuration

```yaml
orchestratorUrl: "http://localhost:8080"
authenticationToken: "dev-token"
logLevel: "debug"
pollingIntervalSeconds: 5
maxConcurrentJobs: 1
stateFile: "./state.json"
tempDir: "./tmp"
```

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
