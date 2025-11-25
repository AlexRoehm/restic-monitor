# Restic Agent Installation Scripts

This directory contains platform-specific installation scripts for the Restic backup agent.

## Platform Support

- **Linux** (systemd): `install-linux.sh`
- **macOS** (launchd): `install-macos.sh`
- **Windows** (Windows Service): `install-windows.ps1`

## Quick Start

### Linux (Ubuntu, Debian, CentOS, RHEL, etc.)

```bash
# Download the agent binary (or build from source)
# Replace with actual download URL or build command
curl -L -o restic-agent https://example.com/restic-agent-linux
chmod +x restic-agent

# Run installation
sudo ./install-linux.sh \
  --orchestrator-url "https://backup.example.com" \
  --auth-token "your-secret-token"
```

### macOS

```bash
# Download the agent binary (or build from source)
curl -L -o restic-agent https://example.com/restic-agent-macos
chmod +x restic-agent

# Run installation
sudo ./install-macos.sh \
  --orchestrator-url "https://backup.example.com" \
  --auth-token "your-secret-token"
```

### Windows

```powershell
# Download the agent binary (or build from source)
# Save as restic-agent.exe

# Run installation (in Administrator PowerShell)
.\install-windows.ps1 `
  -OrchestratorUrl "https://backup.example.com" `
  -AuthToken "your-secret-token"
```

## Installation Options

### Common Options (all platforms)

| Option | Description | Default |
|--------|-------------|---------|
| `orchestrator-url` | Orchestrator server URL | *Required* |
| `auth-token` | Authentication token | *Required* |
| `agent-binary` | Path to agent executable | `./restic-agent` (Windows: `.\restic-agent.exe`) |

### Linux-Specific Options

| Option | Description | Default |
|--------|-------------|---------|
| `--install-dir` | Installation directory | `/usr/local/bin` |
| `--config-dir` | Configuration directory | `/etc/restic-agent` |
| `--state-dir` | State file directory | `/var/lib/restic-agent` |
| `--log-dir` | Log file directory | `/var/log/restic-agent` |
| `--temp-dir` | Temporary directory | `/tmp/restic-agent` |
| `--user` | Service user | `restic-agent` |
| `--group` | Service group | `restic-agent` |

### macOS-Specific Options

| Option | Description | Default |
|--------|-------------|---------|
| `--install-dir` | Installation directory | `/usr/local/bin` |
| `--config-dir` | Configuration directory | `/usr/local/etc/restic-agent` |
| `--state-dir` | State file directory | `/var/lib/restic-agent` |
| `--log-dir` | Log file directory | `/var/log/restic-agent` |
| `--temp-dir` | Temporary directory | `/tmp/restic-agent` |
| `--user` | Service user | Current user |

### Windows-Specific Options

| Option | Description | Default |
|--------|-------------|---------|
| `-InstallDir` | Installation directory | `C:\Program Files\ResticAgent` |
| `-ConfigDir` | Configuration directory | `C:\ProgramData\ResticAgent\config` |
| `-StateDir` | State file directory | `C:\ProgramData\ResticAgent\state` |
| `-LogDir` | Log file directory | `C:\ProgramData\ResticAgent\logs` |
| `-TempDir` | Temporary directory | `C:\ProgramData\ResticAgent\temp` |
| `-ServiceName` | Windows service name | `ResticAgent` |

## Post-Installation

### Verify Installation

After installation, verify the agent is running:

**Linux:**
```bash
sudo systemctl status restic-agent.service
journalctl -u restic-agent.service -f
```

**macOS:**
```bash
sudo launchctl list | grep restic
tail -f /var/log/restic-agent/agent.log
```

**Windows:**
```powershell
Get-Service -Name ResticAgent
Get-Content C:\ProgramData\ResticAgent\logs\agent.log -Tail 50 -Wait
```

### Configuration

The agent configuration is stored in `agent.yaml`:

- **Linux:** `/etc/restic-agent/agent.yaml`
- **macOS:** `/usr/local/etc/restic-agent/agent.yaml`
- **Windows:** `C:\ProgramData\ResticAgent\config\agent.yaml`

See [`../config/agent.example.yaml`](../config/agent.example.yaml) for an example configuration, or [`../docs/agent/configuration-spec.md`](../docs/agent/configuration-spec.md) for the complete specification.

### Managing the Service

**Linux (systemd):**
```bash
sudo systemctl start restic-agent.service    # Start
sudo systemctl stop restic-agent.service     # Stop
sudo systemctl restart restic-agent.service  # Restart
sudo systemctl status restic-agent.service   # Status
sudo systemctl disable restic-agent.service  # Disable auto-start
sudo systemctl enable restic-agent.service   # Enable auto-start
```

**macOS (launchd):**
```bash
sudo launchctl load /Library/LaunchDaemons/com.restic.agent.plist    # Start
sudo launchctl unload /Library/LaunchDaemons/com.restic.agent.plist  # Stop
sudo launchctl list | grep restic                                    # Status
```

**Windows (Service):**
```powershell
Start-Service -Name ResticAgent     # Start
Stop-Service -Name ResticAgent      # Stop
Restart-Service -Name ResticAgent   # Restart
Get-Service -Name ResticAgent       # Status
```

## Uninstallation

### Linux

```bash
sudo systemctl stop restic-agent.service
sudo systemctl disable restic-agent.service
sudo rm /etc/systemd/system/restic-agent.service
sudo systemctl daemon-reload
sudo userdel restic-agent
sudo rm -rf /usr/local/bin/restic-agent
sudo rm -rf /etc/restic-agent
sudo rm -rf /var/lib/restic-agent
sudo rm -rf /var/log/restic-agent
```

### macOS

```bash
sudo launchctl unload /Library/LaunchDaemons/com.restic.agent.plist
sudo rm /Library/LaunchDaemons/com.restic.agent.plist
sudo rm -rf /usr/local/bin/restic-agent
sudo rm -rf /usr/local/etc/restic-agent
sudo rm -rf /var/lib/restic-agent
sudo rm -rf /var/log/restic-agent
```

### Windows

```powershell
Stop-Service -Name ResticAgent
sc.exe delete ResticAgent
Remove-Item -Recurse -Force "C:\Program Files\ResticAgent"
Remove-Item -Recurse -Force "C:\ProgramData\ResticAgent"
```

## Troubleshooting

### Agent Not Starting

1. **Check logs** for error messages
2. **Verify configuration** using the diagnostic tool:
   ```bash
   # Linux/macOS
   /usr/local/bin/restic-agent --test-config /etc/restic-agent/agent.yaml
   
   # Windows
   "C:\Program Files\ResticAgent\restic-agent.exe" --test-config "C:\ProgramData\ResticAgent\config\agent.yaml"
   ```
3. **Test connectivity** to orchestrator:
   ```bash
   curl -v https://backup.example.com/health
   ```
4. **Check permissions** on config and state directories

### Registration Failures

- Verify the `authenticationToken` is correct
- Ensure the orchestrator URL is accessible
- Check firewall rules allow outbound HTTPS

### High CPU/Memory Usage

- Check `maxConcurrentJobs` setting (default: 2)
- Review backup schedules for overlapping jobs
- Check available disk space in `tempDir`

## Security Considerations

1. **Configuration File:** Contains sensitive authentication token
   - Permissions set to `0600` (owner read/write only)
   - Only readable by service user and administrators
   
2. **Service User:** Runs with minimal privileges
   - Linux: Dedicated `restic-agent` user (no login, no home directory)
   - macOS: Regular user account
   - Windows: `NT AUTHORITY\SYSTEM`
   
3. **Network Communication:** Use HTTPS for orchestrator URL
   - Validates TLS certificates
   - Encrypted authentication token transmission
   
4. **State File:** Contains agent ID (UUID)
   - Permissions set to `0600`
   - Stored in protected system directory

## Advanced Configuration

### Custom Installation Path

```bash
# Linux - install to /opt
sudo ./install-linux.sh \
  --orchestrator-url "https://backup.example.com" \
  --auth-token "token" \
  --install-dir "/opt/restic-agent/bin" \
  --config-dir "/opt/restic-agent/etc" \
  --state-dir "/opt/restic-agent/var" \
  --log-dir "/opt/restic-agent/log"
```

### Running as Specific User

```bash
# Linux - run as existing user
sudo ./install-linux.sh \
  --orchestrator-url "https://backup.example.com" \
  --auth-token "token" \
  --user "backup-admin" \
  --group "backup-admin"
```

### Debug Mode

To run the agent with debug logging:

1. Edit the configuration file
2. Set `logLevel: "debug"`
3. Restart the service

## Building from Source

```bash
# Clone the repository
git clone https://github.com/your-org/restic-monitor.git
cd restic-monitor

# Build the agent
go build -o restic-agent ./cmd/restic-agent

# Run installation
sudo ./scripts/install-linux.sh \
  --orchestrator-url "https://backup.example.com" \
  --auth-token "your-token" \
  --agent-binary "./restic-agent"
```

## Support

- Documentation: [`../docs/`](../docs/)
- Configuration Spec: [`../docs/agent/configuration-spec.md`](../docs/agent/configuration-spec.md)
- API Documentation: [`../docs/api/`](../docs/api/)
