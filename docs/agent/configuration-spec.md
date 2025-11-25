# Agent Configuration Specification

## Overview

This document defines the configuration file format (`agent.yaml`) used by the restic-monitor backup agent. The configuration enables the agent to communicate with the orchestrator, authenticate, and operate autonomously.

## Configuration File Location

### Linux
* **System-wide**: `/etc/restic-agent/agent.yaml`
* **User-mode**: `~/.config/restic-agent/agent.yaml`

### macOS
* **System-wide**: `/Library/Application Support/ResticAgent/agent.yaml`
* **User-mode**: `~/Library/Application Support/ResticAgent/agent.yaml`

### Windows
* **System-wide**: `C:\ProgramData\ResticAgent\agent.yaml`
* **User-mode**: `%APPDATA%\ResticAgent\agent.yaml`

### Custom Path
The agent accepts a `--config` flag to override the default location:
```bash
restic-agent --config=/path/to/custom/agent.yaml
```

## Configuration Format

### YAML Schema

```yaml
# Orchestrator connection settings (REQUIRED)
orchestratorUrl: "https://backup.example.com"

# Agent identity (empty on first run, populated after registration)
agentId: ""

# Authentication token (REQUIRED)
authenticationToken: "your-secret-token-here"

# Optional: Override detected hostname
hostnameOverride: ""

# Logging configuration
logLevel: "info"  # debug, info, warn, error
logFile: ""       # Empty = stdout only

# Agent behavior
pollingIntervalSeconds: 30      # How often to check for new tasks
heartbeatIntervalSeconds: 60    # How often to send heartbeat
maxConcurrentJobs: 2            # Max parallel backup jobs

# Network settings
httpTimeoutSeconds: 30
retryMaxAttempts: 3
retryBackoffSeconds: 5

# Local storage
stateFile: ""      # Default: auto-detected based on OS
tempDir: ""        # Default: system temp directory
```

### Minimal Configuration Example

```yaml
orchestratorUrl: "https://backup.example.com"
authenticationToken: "abc123xyz789"
```

All other fields use sensible defaults.

### Full Configuration Example

```yaml
orchestratorUrl: "https://backup.mycompany.com:8080"
agentId: "550e8400-e29b-41d4-a716-446655440000"
authenticationToken: "secure-token-here"
hostnameOverride: "prod-web-01"
logLevel: "debug"
logFile: "/var/log/restic-agent/agent.log"
pollingIntervalSeconds: 15
heartbeatIntervalSeconds: 30
maxConcurrentJobs: 4
httpTimeoutSeconds: 60
retryMaxAttempts: 5
retryBackoffSeconds: 10
stateFile: "/var/lib/restic-agent/state.json"
tempDir: "/var/tmp/restic-agent"
```

## Field Specifications

### orchestratorUrl (REQUIRED)

**Type**: `string`  
**Description**: Base URL of the orchestrator server  
**Validation**:
* Must be valid URL (http:// or https://)
* Must not end with trailing slash
* Port optional (defaults to 80/443)

**Examples**:
```yaml
orchestratorUrl: "https://backup.example.com"
orchestratorUrl: "http://localhost:8080"
orchestratorUrl: "https://10.0.1.50:9000"
```

**Environment Override**: `RESTIC_AGENT_ORCHESTRATOR_URL`

---

### agentId

**Type**: `string` (UUID format)  
**Default**: `""` (empty)  
**Description**: Unique identifier assigned by orchestrator during first registration  
**Validation**:
* Must be valid UUIDv4 or empty string
* Automatically populated after first registration
* Should not be manually edited

**Behavior**:
* **Empty**: Agent will register with orchestrator on first start
* **Set**: Agent uses this ID for all API calls

**Examples**:
```yaml
agentId: ""  # Before registration
agentId: "550e8400-e29b-41d4-a716-446655440000"  # After registration
```

**Environment Override**: Not supported (managed by agent)

---

### authenticationToken (REQUIRED)

**Type**: `string`  
**Description**: Authentication token for orchestrator API access  
**Validation**:
* Must not be empty
* Minimum length: 16 characters (recommended: 32+)

**Security Notes**:
* Store securely with appropriate file permissions (600 on Unix)
* Rotate periodically
* Never commit to version control

**Examples**:
```yaml
authenticationToken: "9a8f7e6d5c4b3a2109876543210abcdef"
```

**Environment Override**: `RESTIC_AGENT_AUTH_TOKEN`

---

### hostnameOverride

**Type**: `string`  
**Default**: `""` (use system hostname)  
**Description**: Override the automatically detected hostname  
**Validation**:
* Must be valid hostname (RFC 1123)
* Max length: 253 characters
* Allowed characters: alphanumeric, hyphens, periods

**Use Cases**:
* Multiple agents on same host (different roles)
* Friendly names instead of technical hostnames
* Containerized environments with random hostnames

**Examples**:
```yaml
hostnameOverride: "prod-web-01"
hostnameOverride: "backup-agent-east"
```

**Environment Override**: `RESTIC_AGENT_HOSTNAME`

---

### logLevel

**Type**: `string`  
**Default**: `"info"`  
**Description**: Minimum log level to output  
**Allowed Values**: `debug`, `info`, `warn`, `error`  
**Validation**: Must be one of allowed values (case-insensitive)

**Log Level Behaviors**:
* **debug**: All messages including verbose diagnostics
* **info**: Normal operational messages (default)
* **warn**: Warning messages and above
* **error**: Only error messages

**Examples**:
```yaml
logLevel: "info"    # Production default
logLevel: "debug"   # Troubleshooting
logLevel: "error"   # Minimal logging
```

**Environment Override**: `RESTIC_AGENT_LOG_LEVEL`

---

### logFile

**Type**: `string`  
**Default**: `""` (stdout only)  
**Description**: Path to log file for persistent logging  
**Validation**:
* Must be absolute path or empty
* Parent directory must exist or be creatable
* Agent must have write permissions

**Behavior**:
* **Empty**: Logs only to stdout (captured by systemd/launchd/Windows service)
* **Set**: Logs to both file and stdout
* Log rotation handled externally (logrotate, etc.)

**Examples**:
```yaml
logFile: "/var/log/restic-agent/agent.log"           # Linux
logFile: "C:\\ProgramData\\ResticAgent\\agent.log"   # Windows
logFile: "/Library/Logs/ResticAgent/agent.log"       # macOS
```

**Environment Override**: `RESTIC_AGENT_LOG_FILE`

---

### pollingIntervalSeconds

**Type**: `integer`  
**Default**: `30`  
**Description**: How often (in seconds) the agent checks for new backup tasks  
**Validation**:
* Minimum: `5` seconds
* Maximum: `3600` seconds (1 hour)
* Must be positive integer

**Performance Notes**:
* Lower values = faster task pickup, more network traffic
* Higher values = reduced load, slower response time
* Recommended range: 15-60 seconds

**Examples**:
```yaml
pollingIntervalSeconds: 30   # Default
pollingIntervalSeconds: 15   # Faster response
pollingIntervalSeconds: 300  # Low-traffic mode
```

**Environment Override**: `RESTIC_AGENT_POLLING_INTERVAL`

---

### heartbeatIntervalSeconds

**Type**: `integer`  
**Default**: `60`  
**Description**: How often (in seconds) the agent sends heartbeat to orchestrator  
**Validation**:
* Minimum: `10` seconds
* Maximum: `600` seconds (10 minutes)
* Must be positive integer
* Should be >= pollingIntervalSeconds

**Purpose**:
* Updates agent status (online/offline)
* Reports system metrics (CPU, memory, disk)
* Keeps connection alive

**Examples**:
```yaml
heartbeatIntervalSeconds: 60   # Default
heartbeatIntervalSeconds: 30   # High-availability mode
heartbeatIntervalSeconds: 300  # Low-frequency mode
```

**Environment Override**: `RESTIC_AGENT_HEARTBEAT_INTERVAL`

---

### maxConcurrentJobs

**Type**: `integer`  
**Default**: `2`  
**Description**: Maximum number of backup jobs to run simultaneously  
**Validation**:
* Minimum: `1`
* Maximum: `10`
* Must be positive integer

**Resource Considerations**:
* Each job consumes CPU, memory, disk I/O, network bandwidth
* Consider available system resources
* Typical settings: 1-4 for desktops, 2-8 for servers

**Examples**:
```yaml
maxConcurrentJobs: 2   # Default
maxConcurrentJobs: 1   # Resource-constrained systems
maxConcurrentJobs: 4   # High-performance servers
```

**Environment Override**: `RESTIC_AGENT_MAX_JOBS`

---

### httpTimeoutSeconds

**Type**: `integer`  
**Default**: `30`  
**Description**: HTTP request timeout for orchestrator API calls  
**Validation**:
* Minimum: `5` seconds
* Maximum: `300` seconds (5 minutes)
* Must be positive integer

**Use Cases**:
* Increase for slow networks
* Increase for large policy downloads
* Decrease for fast failure detection

**Examples**:
```yaml
httpTimeoutSeconds: 30   # Default
httpTimeoutSeconds: 60   # Slow networks
httpTimeoutSeconds: 10   # Fast failure detection
```

**Environment Override**: `RESTIC_AGENT_HTTP_TIMEOUT`

---

### retryMaxAttempts

**Type**: `integer`  
**Default**: `3`  
**Description**: Maximum number of retry attempts for failed API calls  
**Validation**:
* Minimum: `1` (no retries)
* Maximum: `10`
* Must be positive integer

**Retry Logic**:
* Exponential backoff between retries
* Only retries on network errors, not HTTP 4xx errors
* Total time = timeout × attempts × backoff multiplier

**Examples**:
```yaml
retryMaxAttempts: 3   # Default
retryMaxAttempts: 1   # No retries
retryMaxAttempts: 5   # Aggressive retries
```

**Environment Override**: `RESTIC_AGENT_RETRY_MAX`

---

### retryBackoffSeconds

**Type**: `integer`  
**Default**: `5`  
**Description**: Initial backoff delay (in seconds) before first retry  
**Validation**:
* Minimum: `1` second
* Maximum: `60` seconds
* Must be positive integer

**Backoff Strategy**:
* First retry: wait `retryBackoffSeconds`
* Second retry: wait `retryBackoffSeconds × 2`
* Third retry: wait `retryBackoffSeconds × 4`
* Exponential backoff with jitter

**Examples**:
```yaml
retryBackoffSeconds: 5    # Default
retryBackoffSeconds: 1    # Fast retries
retryBackoffSeconds: 10   # Slow retries
```

**Environment Override**: `RESTIC_AGENT_RETRY_BACKOFF`

---

### stateFile

**Type**: `string`  
**Default**: `""` (auto-detected)  
**Description**: Path to persistent state file storing agent identity  
**Validation**:
* Must be absolute path or empty
* Parent directory must exist or be creatable
* Agent must have write permissions

**Default Locations** (when empty):
* **Linux**: `/var/lib/restic-agent/state.json` (root) or `~/.local/share/restic-agent/state.json` (user)
* **macOS**: `~/Library/Application Support/ResticAgent/state.json`
* **Windows**: `C:\ProgramData\ResticAgent\state.json` or `%APPDATA%\ResticAgent\state.json`

**State File Contents**:
```json
{
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "orchestratorUrl": "https://backup.example.com",
  "registeredAt": "2025-11-24T14:30:00Z"
}
```

**Examples**:
```yaml
stateFile: "/var/lib/restic-agent/state.json"
stateFile: "C:\\ProgramData\\ResticAgent\\state.json"
```

**Environment Override**: `RESTIC_AGENT_STATE_FILE`

---

### tempDir

**Type**: `string`  
**Default**: `""` (system temp directory)  
**Description**: Directory for temporary files during backup operations  
**Validation**:
* Must be absolute path or empty
* Must have sufficient free space (>1GB recommended)
* Agent must have write permissions

**Default Locations** (when empty):
* **Linux**: `/tmp/restic-agent` or `$TMPDIR`
* **macOS**: `/tmp/restic-agent` or `$TMPDIR`
* **Windows**: `%TEMP%\ResticAgent`

**Use Cases**:
* Custom location for dedicated backup volume
* SSD for better performance
* Network mount for distributed temp storage

**Examples**:
```yaml
tempDir: "/var/tmp/restic-agent"
tempDir: "D:\\Temp\\ResticAgent"
tempDir: "/mnt/fast-ssd/restic-temp"
```

**Environment Override**: `RESTIC_AGENT_TEMP_DIR`

---

## Environment Variable Override Rules

### Priority Order (highest to lowest):

1. **Command-line flags** (e.g., `--config`, `--log-level`)
2. **Environment variables** (e.g., `RESTIC_AGENT_LOG_LEVEL`)
3. **Configuration file values** (`agent.yaml`)
4. **Default values**

### Supported Environment Variables

| Variable | Config Field | Example |
|----------|--------------|---------|
| `RESTIC_AGENT_ORCHESTRATOR_URL` | `orchestratorUrl` | `https://backup.example.com` |
| `RESTIC_AGENT_AUTH_TOKEN` | `authenticationToken` | `abc123xyz` |
| `RESTIC_AGENT_HOSTNAME` | `hostnameOverride` | `prod-web-01` |
| `RESTIC_AGENT_LOG_LEVEL` | `logLevel` | `debug` |
| `RESTIC_AGENT_LOG_FILE` | `logFile` | `/var/log/agent.log` |
| `RESTIC_AGENT_POLLING_INTERVAL` | `pollingIntervalSeconds` | `30` |
| `RESTIC_AGENT_HEARTBEAT_INTERVAL` | `heartbeatIntervalSeconds` | `60` |
| `RESTIC_AGENT_MAX_JOBS` | `maxConcurrentJobs` | `2` |
| `RESTIC_AGENT_HTTP_TIMEOUT` | `httpTimeoutSeconds` | `30` |
| `RESTIC_AGENT_RETRY_MAX` | `retryMaxAttempts` | `3` |
| `RESTIC_AGENT_RETRY_BACKOFF` | `retryBackoffSeconds` | `5` |
| `RESTIC_AGENT_STATE_FILE` | `stateFile` | `/var/lib/state.json` |
| `RESTIC_AGENT_TEMP_DIR` | `tempDir` | `/tmp/restic` |

### Example: Environment Variable Override

```bash
# Override log level via environment
export RESTIC_AGENT_LOG_LEVEL=debug
export RESTIC_AGENT_POLLING_INTERVAL=15

# Start agent (will use environment values)
restic-agent
```

---

## Validation Rules

### On Agent Startup

The agent performs the following validation:

1. **Config file exists**: Error if `--config` specified but file missing
2. **YAML syntax valid**: Error on parse failure with line number
3. **Required fields present**: `orchestratorUrl`, `authenticationToken`
4. **Field types correct**: String fields are strings, integers are integers
5. **Value ranges valid**: Intervals within min/max bounds
6. **URL format valid**: `orchestratorUrl` is parseable URL
7. **Hostname valid**: `hostnameOverride` matches RFC 1123 if set

### Validation Errors

Example error messages:

```
ERROR: Configuration validation failed:
  - Field 'orchestratorUrl' is required but missing
  - Field 'pollingIntervalSeconds' must be between 5 and 3600 (got: 0)
  - Field 'logLevel' must be one of: debug, info, warn, error (got: 'verbose')
```

---

## Security Considerations

### File Permissions

**Unix/Linux/macOS**:
```bash
chmod 600 /etc/restic-agent/agent.yaml  # Read/write owner only
chown restic-agent:restic-agent /etc/restic-agent/agent.yaml
```

**Windows**:
```powershell
icacls "C:\ProgramData\ResticAgent\agent.yaml" /inheritance:r /grant:r "NT AUTHORITY\SYSTEM:(F)" "Administrators:(F)"
```

### Token Security

* **Never commit** `authenticationToken` to version control
* **Use secrets management**: Environment variables or vault systems for production
* **Rotate regularly**: Update tokens every 90 days minimum
* **Audit access**: Log all token usage for security monitoring

### Example: Using Environment Variable for Token

```yaml
orchestratorUrl: "https://backup.example.com"
# authenticationToken loaded from RESTIC_AGENT_AUTH_TOKEN env var
logLevel: "info"
```

```bash
export RESTIC_AGENT_AUTH_TOKEN="secure-token-from-vault"
restic-agent
```

---

## Migration & Compatibility

### Backward Compatibility

* New fields added with sensible defaults (non-breaking)
* Deprecated fields warned but still functional for 1 major version
* Breaking changes only in major version updates

### Version-Specific Behavior

Configuration format version can be specified (optional):

```yaml
version: 1
orchestratorUrl: "https://backup.example.com"
authenticationToken: "token"
```

If omitted, latest version assumed.

---

## Troubleshooting

### Agent Won't Start

**Check configuration**:
```bash
restic-agent --self-test --config=/etc/restic-agent/agent.yaml
```

**Validate YAML syntax**:
```bash
yamllint /etc/restic-agent/agent.yaml
```

**Check file permissions**:
```bash
ls -l /etc/restic-agent/agent.yaml
# Should show: -rw------- (600)
```

### Environment Variables Not Working

**Verify environment**:
```bash
env | grep RESTIC_AGENT
```

**Check priority order**: Command-line flags override environment variables

**Debug mode**:
```bash
restic-agent --log-level=debug
# Will show: "Loaded config from /etc/restic-agent/agent.yaml"
# Will show: "Environment override: RESTIC_AGENT_LOG_LEVEL=debug"
```

---

## Complete Working Example

**File**: `/etc/restic-agent/agent.yaml`

```yaml
# Restic Monitor Agent Configuration
# Documentation: https://docs.restic-monitor.example.com/agent/config

# Required: Orchestrator connection
orchestratorUrl: "https://backup.mycompany.com"
authenticationToken: "9a8f7e6d5c4b3a2109876543210abcdef"

# Agent identity (populated after first registration)
agentId: "550e8400-e29b-41d4-a716-446655440000"

# Optional: Override hostname
# hostnameOverride: "prod-web-01"

# Logging
logLevel: "info"
logFile: "/var/log/restic-agent/agent.log"

# Operational settings
pollingIntervalSeconds: 30
heartbeatIntervalSeconds: 60
maxConcurrentJobs: 2

# Network tuning
httpTimeoutSeconds: 30
retryMaxAttempts: 3
retryBackoffSeconds: 5

# Storage paths
stateFile: "/var/lib/restic-agent/state.json"
tempDir: "/var/tmp/restic-agent"
```

**Start agent**:
```bash
sudo systemctl start restic-agent
sudo journalctl -u restic-agent -f
```

Expected logs:
```
INFO: Starting restic-monitor agent v1.0.0
INFO: Loaded configuration from /etc/restic-agent/agent.yaml
INFO: Agent ID: 550e8400-e29b-41d4-a716-446655440000
INFO: Orchestrator: https://backup.mycompany.com
INFO: Heartbeat interval: 60s, Polling interval: 30s
INFO: Agent started successfully
```
