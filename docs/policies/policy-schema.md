# Policy Schema

## Overview

Backup policies define **what** to back up, **when** to back it up, and **how** to store it. Each policy contains a schedule (cron expression), paths to include/exclude, repository configuration, and optional settings for performance and retention.

Policies are created and managed by operators through the orchestrator API and are later assigned to agents for execution.

## Table of Contents

1. [Core Schema](#core-schema)
2. [Field Definitions](#field-definitions)
3. [Repository Types](#repository-types)
4. [Retention Rules](#retention-rules)
5. [Validation Rules](#validation-rules)
6. [Examples](#examples)
7. [JSON Schema](#json-schema)

---

## Core Schema

### Policy Object

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "daily-documents-backup",
  "description": "Daily backup of user documents to S3",
  "schedule": "0 2 * * *",
  "includePaths": [
    "/home/user/Documents",
    "/home/user/Pictures"
  ],
  "excludePaths": [
    "*/node_modules",
    "*/.git",
    "*.tmp"
  ],
  "repository": {
    "type": "s3",
    "bucket": "my-backups",
    "prefix": "restic/documents",
    "region": "us-west-2"
  },
  "retention": {
    "keepLast": 7,
    "keepDaily": 14,
    "keepWeekly": 8,
    "keepMonthly": 12,
    "keepYearly": 5
  },
  "bandwidthLimitKBps": 10240,
  "parallelFiles": 4,
  "createdAt": "2025-11-24T10:00:00Z",
  "updatedAt": "2025-11-24T10:00:00Z"
}
```

---

## Field Definitions

### Required Fields

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | Server-generated, immutable | Unique identifier for the policy |
| `name` | string | 3-100 chars, unique, alphanumeric + `-_` | Human-readable policy name |
| `schedule` | string | Valid cron expression | When to run backups (UTC timezone) |
| `includePaths` | array[string] | Min 1 item, max 100 items | Paths to include in backup |
| `repository` | object | See [Repository Types](#repository-types) | Backup destination configuration |

### Optional Fields

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `description` | string | Max 500 chars | Human-readable description |
| `excludePaths` | array[string] | Max 1000 items | Paths/patterns to exclude from backup |
| `retention` | object | See [Retention Rules](#retention-rules) | Snapshot retention policy |
| `bandwidthLimitKBps` | integer | > 0, max 1000000 | Upload bandwidth limit in KB/s |
| `parallelFiles` | integer | 1-32 | Number of files to process in parallel |

### Metadata Fields (Read-Only)

| Field | Type | Description |
|-------|------|-------------|
| `createdAt` | timestamp | When the policy was created (RFC3339) |
| `updatedAt` | timestamp | When the policy was last modified (RFC3339) |

---

## Repository Types

Policies support multiple repository backend types. Each type requires specific configuration fields.

### S3 Repository

```json
{
  "type": "s3",
  "bucket": "my-backup-bucket",
  "prefix": "restic/server1",
  "region": "us-west-2",
  "endpoint": "https://s3.us-west-2.amazonaws.com",
  "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
  "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
```

**Required Fields**:
- `type`: Must be `"s3"`
- `bucket`: S3 bucket name
- `prefix`: Path prefix within bucket (recommended for organization)

**Optional Fields**:
- `region`: AWS region (default: `us-east-1`)
- `endpoint`: Custom S3 endpoint (for S3-compatible services)
- `accessKeyId`: AWS access key (can be provided via agent environment)
- `secretAccessKey`: AWS secret key (can be provided via agent environment)

### REST Server Repository

```json
{
  "type": "rest-server",
  "url": "https://backup.example.com:8000/user1",
  "username": "backup-user",
  "password": "secure-password"
}
```

**Required Fields**:
- `type`: Must be `"rest-server"`
- `url`: Full URL to the REST server repository

**Optional Fields**:
- `username`: HTTP Basic Auth username
- `password`: HTTP Basic Auth password
- `tlsClientCert`: Path to TLS client certificate

### Filesystem Repository

```json
{
  "type": "fs",
  "path": "/mnt/backup/restic-repo"
}
```

**Required Fields**:
- `type`: Must be `"fs"`
- `path`: Absolute path to the repository directory

**Optional Fields**:
- None

### SFTP Repository

```json
{
  "type": "sftp",
  "host": "backup.example.com",
  "user": "backup-user",
  "path": "/backups/restic",
  "port": 22
}
```

**Required Fields**:
- `type`: Must be `"sftp"`
- `host`: SFTP server hostname
- `user`: SSH username
- `path`: Remote path to repository

**Optional Fields**:
- `port`: SSH port (default: 22)
- `keyFile`: Path to SSH private key

---

## Retention Rules

Retention rules determine how long snapshots are kept. All fields are optional; if no retention is specified, snapshots are kept indefinitely.

```json
{
  "keepLast": 7,
  "keepHourly": 24,
  "keepDaily": 14,
  "keepWeekly": 8,
  "keepMonthly": 12,
  "keepYearly": 5,
  "keepWithin": "30d"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `keepLast` | integer | Keep the last N snapshots |
| `keepHourly` | integer | Keep one snapshot per hour for the last N hours |
| `keepDaily` | integer | Keep one snapshot per day for the last N days |
| `keepWeekly` | integer | Keep one snapshot per week for the last N weeks |
| `keepMonthly` | integer | Keep one snapshot per month for the last N months |
| `keepYearly` | integer | Keep one snapshot per year for the last N years |
| `keepWithin` | string | Keep all snapshots within duration (e.g., "30d", "12h") |

**Validation**:
- All values must be positive integers
- At least one retention rule should be specified (recommended)
- Multiple rules can be combined

---

## Validation Rules

### Name Validation

```
- Required: true
- Min length: 3 characters
- Max length: 100 characters
- Pattern: ^[a-zA-Z0-9_-]+$
- Uniqueness: Must be unique across all policies
- Case sensitivity: Case-sensitive
```

### Schedule Validation

```
- Required: true
- Format: Cron expression (5 or 6 fields)
- Examples:
  * "0 2 * * *" - Daily at 2:00 AM
  * "*/30 * * * *" - Every 30 minutes
  * "0 0 * * 0" - Weekly on Sunday at midnight
- Must be parseable by cron parser
- Timezone: All schedules interpreted as UTC
```

### Include Paths Validation

```
- Required: true
- Min items: 1
- Max items: 100
- Each path:
  * Max length: 4096 characters
  * Must be absolute path (start with / or drive letter on Windows)
  * No duplicate paths
```

### Exclude Paths Validation

```
- Required: false
- Max items: 1000
- Supports glob patterns:
  * "*.tmp" - All .tmp files
  * "*/node_modules" - All node_modules directories
  * ".git" - All .git directories
```

### Repository Type Validation

```
- Required: true
- Allowed values: "s3", "rest-server", "fs", "sftp"
- Must include all required fields for the selected type
- Type-specific validation:
  * S3: bucket name must follow AWS naming rules
  * REST: URL must be valid HTTP/HTTPS
  * FS: path must be absolute
  * SFTP: host must be valid hostname/IP
```

### Bandwidth Limit Validation

```
- Required: false
- Min value: 1 KB/s
- Max value: 1000000 KB/s (1 GB/s)
- Unit: KB/s (kilobytes per second)
```

### Parallel Files Validation

```
- Required: false
- Min value: 1
- Max value: 32
- Recommended: 2-8 for most use cases
```

---

## Examples

### Example 1: Simple Daily Backup to S3

```json
{
  "name": "daily-home-backup",
  "description": "Backup home directory daily",
  "schedule": "0 3 * * *",
  "includePaths": ["/home/user"],
  "excludePaths": [
    "*/Downloads",
    "*/.cache",
    "*.tmp"
  ],
  "repository": {
    "type": "s3",
    "bucket": "my-backups",
    "prefix": "home",
    "region": "us-west-2"
  },
  "retention": {
    "keepDaily": 7,
    "keepWeekly": 4,
    "keepMonthly": 6
  }
}
```

### Example 2: High-Frequency Backup to REST Server

```json
{
  "name": "database-backup",
  "description": "Database backup every 4 hours",
  "schedule": "0 */4 * * *",
  "includePaths": ["/var/lib/postgresql/data"],
  "repository": {
    "type": "rest-server",
    "url": "https://backup.example.com:8000/postgres"
  },
  "retention": {
    "keepLast": 10,
    "keepDaily": 14
  },
  "parallelFiles": 1
}
```

### Example 3: Weekly Backup to Local Filesystem

```json
{
  "name": "weekly-archive",
  "description": "Weekly full system backup",
  "schedule": "0 1 * * 0",
  "includePaths": [
    "/",
    "/home"
  ],
  "excludePaths": [
    "/tmp",
    "/var/tmp",
    "/proc",
    "/sys",
    "/dev"
  ],
  "repository": {
    "type": "fs",
    "path": "/mnt/backup/weekly"
  },
  "retention": {
    "keepWeekly": 8,
    "keepMonthly": 12
  }
}
```

### Example 4: Bandwidth-Limited Backup

```json
{
  "name": "limited-bandwidth-backup",
  "description": "Backup with bandwidth constraints",
  "schedule": "0 2 * * *",
  "includePaths": ["/data"],
  "repository": {
    "type": "s3",
    "bucket": "backups",
    "prefix": "data"
  },
  "bandwidthLimitKBps": 5120,
  "parallelFiles": 2,
  "retention": {
    "keepLast": 7
  }
}
```

---

## JSON Schema

### Formal JSON Schema Definition

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "BackupPolicy",
  "type": "object",
  "required": ["name", "schedule", "includePaths", "repository"],
  "properties": {
    "id": {
      "type": "string",
      "format": "uuid",
      "readOnly": true
    },
    "name": {
      "type": "string",
      "minLength": 3,
      "maxLength": 100,
      "pattern": "^[a-zA-Z0-9_-]+$"
    },
    "description": {
      "type": "string",
      "maxLength": 500
    },
    "schedule": {
      "type": "string",
      "pattern": "^(@(annually|yearly|monthly|weekly|daily|hourly))|(((\\*|\\?|[0-9]+(,|-|/)?)+\\s*){5,6})$"
    },
    "includePaths": {
      "type": "array",
      "minItems": 1,
      "maxItems": 100,
      "items": {
        "type": "string",
        "maxLength": 4096
      }
    },
    "excludePaths": {
      "type": "array",
      "maxItems": 1000,
      "items": {
        "type": "string",
        "maxLength": 4096
      }
    },
    "repository": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "enum": ["s3", "rest-server", "fs", "sftp"]
        }
      },
      "oneOf": [
        {
          "properties": {
            "type": { "const": "s3" },
            "bucket": { "type": "string" },
            "prefix": { "type": "string" },
            "region": { "type": "string" },
            "endpoint": { "type": "string", "format": "uri" },
            "accessKeyId": { "type": "string" },
            "secretAccessKey": { "type": "string" }
          },
          "required": ["bucket"]
        },
        {
          "properties": {
            "type": { "const": "rest-server" },
            "url": { "type": "string", "format": "uri" },
            "username": { "type": "string" },
            "password": { "type": "string" }
          },
          "required": ["url"]
        },
        {
          "properties": {
            "type": { "const": "fs" },
            "path": { "type": "string" }
          },
          "required": ["path"]
        },
        {
          "properties": {
            "type": { "const": "sftp" },
            "host": { "type": "string" },
            "user": { "type": "string" },
            "path": { "type": "string" },
            "port": { "type": "integer", "minimum": 1, "maximum": 65535 },
            "keyFile": { "type": "string" }
          },
          "required": ["host", "user", "path"]
        }
      ]
    },
    "retention": {
      "type": "object",
      "properties": {
        "keepLast": { "type": "integer", "minimum": 1 },
        "keepHourly": { "type": "integer", "minimum": 1 },
        "keepDaily": { "type": "integer", "minimum": 1 },
        "keepWeekly": { "type": "integer", "minimum": 1 },
        "keepMonthly": { "type": "integer", "minimum": 1 },
        "keepYearly": { "type": "integer", "minimum": 1 },
        "keepWithin": { "type": "string", "pattern": "^[0-9]+(h|d|w|m|y)$" }
      },
      "minProperties": 1
    },
    "bandwidthLimitKBps": {
      "type": "integer",
      "minimum": 1,
      "maximum": 1000000
    },
    "parallelFiles": {
      "type": "integer",
      "minimum": 1,
      "maximum": 32
    },
    "createdAt": {
      "type": "string",
      "format": "date-time",
      "readOnly": true
    },
    "updatedAt": {
      "type": "string",
      "format": "date-time",
      "readOnly": true
    }
  }
}
```

---

## Usage Notes

### Creating Policies

When creating a policy via `POST /policies`, the following fields are ignored/server-generated:
- `id` - Generated by server
- `createdAt` - Set to current timestamp
- `updatedAt` - Set to current timestamp

### Updating Policies

When updating a policy via `PUT /policies/{id}`, the following fields cannot be changed:
- `id` - Immutable
- `createdAt` - Immutable

The `updatedAt` field is automatically updated to the current timestamp.

### Deleting Policies

Policies that are currently assigned to agents cannot be deleted. The API will return a `409 Conflict` error. Remove all agent assignments before deletion.

### Repository Credentials

While repository credentials (like S3 access keys or passwords) can be stored in the policy, it's recommended to:
1. Use environment variables on the agent side
2. Use instance profiles (for AWS)
3. Use secrets management systems

The orchestrator will pass repository configuration to agents, but agents should prioritize environment-based credentials.

---

## Changelog

### v1.0 (Current)
- Initial policy schema definition
- Support for S3, REST server, filesystem, and SFTP repositories
- Cron-based scheduling
- Path inclusion/exclusion with glob patterns
- Retention rules
- Performance options (bandwidth, parallelism)
