# Restic Monitor

This service periodically inspects Restic repositories, stores metadata via GORM, and exposes a small HTTP API for monitoring.

## Requirements

- Go 1.22+
- `restic` CLI available in `PATH`
- Access to the target Restic repositories (set by the configuration below)

## Configuration

Configure the monitor through environment variables (defaults in parentheses):

| Variable | Description | Default |
| --- | --- | --- |
| `RESTIC_REPOSITORY` | Restic repo to check (required if not seeding with `TARGETS_FILE`) | `` |
| `RESTIC_PASSWORD` | Repository password | `` |
| `RESTIC_PASSWORD_FILE` | File that contains the password | `` |
| `RESTIC_CERT_FILE` | Certificate for TLS backend (`restic check`/S3) | `` |
| `RESTIC_BINARY` | Path to restic binary | `restic` |
| `CHECK_INTERVAL` | Interval between checks | `5m` |
| `DATABASE_DSN` | SQLite DSN | `restic-monitor.db` |
| `API_LISTEN_ADDR` | HTTP listen address | `:8080` |
| `SNAPSHOT_FILE_LIMIT` | Number of files to persist per snapshot | `200` |
| `TARGETS_FILE` | JSON file that seeds the `restic_targets` table | `examples/targets.example.json` |

### Targets configuration

Create a JSON file with one entry per Restic repository you want to monitor. Each entry can provide a `name`, `repository`, `password`, `password_file`, and `certificate_file`. The default path is `targets.json`, but you can override it with `TARGETS_FILE`.

```json
[
  {
    "name": "home",
    "repository": "/path/to/home-repo",
    "password_file": "/etc/restic/home.pass"
  },
  {
    "name": "work",
    "repository": "s3:https://example.com/restic",
    "password": "supersecret",
    "certificate_file": "/etc/restic/work-ca.pem"
  }
]
```

The monitor will seed the `restic_targets` table from this file at startup and reapply the definitions on every run. Changing passwords or repository URIs in the file updates the corresponding database rows. Use the `examples/targets.example.json` file as a starting point.

## Running

```bash
source .env.example
RESTIC_REPOSITORY=/path/to/repo \
RESTIC_PASSWORD_FILE=/etc/restic/pass \
RESTIC_CERT_FILE=/etc/restic/custom-ca.pem \
go run ./cmd/restic-monitor
```

The monitor will:
- Record the latest snapshot timestamp, snapshot count, health, and file list for every configured repository.
- Run `restic check --json` to determine repository health.
- Persist the results via GORM models under `internal/store` and `restic_targets`.
- Start an HTTP API on `/status` (optionally filtered with `?name=<target-name>`) that returns the stored status.

## Testing

```bash
go test ./...
```
