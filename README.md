# Restic Backup Monitor

A beautiful web-based monitoring dashboard for Restic backup repositories. Monitor multiple backup targets, check their health status, and unlock locked repositories - all from a modern Vue.js interface.

## Features

- 📊 **Beautiful Dashboard** - Modern UI built with Vue 3, Tailwind CSS, and DaisyUI
- 🔍 **Multi-Target Monitoring** - Monitor multiple Restic repositories simultaneously
- ✅ **Health Checks** - Automatic repository health validation
- 🔓 **Repository Unlock** - Unlock locked repositories with one click
- 🌍 **Internationalization** - Built-in English and German translations
- 🎨 **Dark Mode** - Light and dark theme support
- 📱 **Responsive Design** - Works on desktop, tablet, and mobile
- 🔄 **Auto-Refresh** - Real-time status updates every 30 seconds
- 🐳 **Docker Support** - Easy deployment with Docker and Docker Compose

## Quick Start with Docker

### Docker Compose (Recommended)

1. Create a `config/targets.json` file:

```json
[
  {
    "name": "production-db",
    "repository": "rest:https://backup.example.com/prod-db",
    "password": "your-restic-password"
  },
  {
    "name": "home-server",
    "repository": "/mnt/backup/home",
    "passwordFile": "/secrets/restic-password.txt"
  }
]
```

2. Run with Docker Compose:

```bash
docker-compose up -d
```

3. Open http://localhost:8080 in your browser

### Docker Run

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config \
  --name restic-monitor \
  guxxde/restic-monitor:latest
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_DSN` | `/app/data/restic-monitor.db` | SQLite database path |
| `TARGETS_FILE` | `/app/config/targets.json` | Path to targets configuration file |
| `API_LISTEN_ADDR` | `:8080` | API server listen address |
| `CHECK_INTERVAL` | `5m` | Interval between backup checks |
| `RESTIC_TIMEOUT` | `2m` | Timeout for restic CLI commands |
| `SNAPSHOT_FILE_LIMIT` | `200` | Maximum number of files to list per snapshot |
| `RESTIC_BINARY` | `restic` | Path to restic binary |
| `AUTH_USERNAME` | _(empty)_ | Basic auth username (optional, leave empty to disable auth) |
| `AUTH_PASSWORD` | _(empty)_ | Basic auth password (optional, leave empty to disable auth) |
| `AUTH_TOKEN` | _(empty)_ | API bearer token (optional, alternative to username/password) |

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
- `password` - Restic repository password (optional if using passwordFile)
- `passwordFile` - Path to file containing password (optional if using password)
- `certificateFile` - Path to CA certificate for HTTPS repositories (optional)

### Supported Repository Types

- Local: `/path/to/repo`
- SFTP: `sftp:user@host:/path/to/repo`
- REST: `rest:https://user:pass@host:8000/`
- S3: `s3:s3.amazonaws.com/bucket/path`
- Azure: `azure:container:/path`
- B2: `b2:bucketname:/path`
- And more - see [Restic documentation](https://restic.readthedocs.io/)

## Development

### Prerequisites

- Go 1.22 or later
- Node.js 20 or later
- Restic CLI

### Build from Source

1. Clone the repository:
```bash
git clone https://github.com/guxxde/restic-monitor.git
cd restic-monitor
```

2. Build frontend:
```bash
cd frontend
npm install
npm run build
cd ..
```

3. Build backend:
```bash
go mod download
go build -o restic-monitor cmd/restic-monitor/main.go
```

4. Create configuration:
```bash
mkdir -p config data
cp .env.example .env
# Edit .env and config/targets.json
```

5. Run:
```bash
export $(cat .env | grep -v '^#' | xargs)
./restic-monitor
```

### Development Mode

Backend:
```bash
export $(cat .env | grep -v '^#' | xargs)
go run cmd/restic-monitor/main.go
```

Frontend:
```bash
cd frontend
npm run dev
```

The frontend dev server runs on http://localhost:5173 and proxies API requests to the backend on http://localhost:8080.

## API Endpoints

### GET `/status`

Get status of all targets or a specific target.

**Query Parameters:**
- `name` (optional) - Filter by target name

**Response:**
```json
[
  {
    "name": "production-db",
    "repository": "rest:https://backup.example.com/prod-db",
    "latestBackup": "2025-11-20T10:30:00Z",
    "snapshotCount": 42,
    "health": true,
    "statusMessage": "",
    "checkedAt": "2025-11-20T15:45:00Z",
    "files": [...]
  }
]
```

### POST `/unlock/{name}`

Unlock a locked repository.

**Response:**
```json
{
  "status": "unlocked",
  "target": "production-db",
  "message": "successfully removed locks"
}
```

## Architecture

- **Backend**: Go with GORM (SQLite), periodic Restic command execution
- **Frontend**: Vue 3, Tailwind CSS v4, DaisyUI, vue-i18n
- **Database**: SQLite (embedded, no external DB required)
- **API**: RESTful HTTP API with CORS support

## Security Considerations

- Store passwords securely using `passwordFile` instead of plain text
- Use HTTPS for remote repositories
- Validate certificate files for TLS connections
- Run container as non-root user (UID 1000)
- Mount sensitive files read-only in Docker

## GitHub Actions Setup

To enable automatic Docker image builds:

1. Go to your repository Settings → Secrets and variables → Actions
2. Add the following secrets:
   - `DOCKER_USERNAME` - Your Docker Hub username
   - `DOCKER_PASSWORD` - Your Docker Hub password or access token

The workflow will automatically:
- Build multi-arch images (amd64/arm64) on push to main
- Tag images with version tags on releases
- Push to Docker Hub as `guxxde/restic-monitor`

## Testing

```bash
go test ./...
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

- [GitHub Issues](https://github.com/guxxde/restic-monitor/issues)
- [Restic Documentation](https://restic.readthedocs.io/)
