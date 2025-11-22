# Restic Monitor API Documentation

This directory contains the OpenAPI (Swagger) specification for the Restic Monitor API.

## Viewing the Documentation

### Option 1: Built-in Swagger UI (Recommended)
If `SHOW_SWAGGER=true` is set in your environment:
```bash
# Set in .env file
SHOW_SWAGGER=true

# Then visit
http://localhost:8080/api/v1/swagger
```

The Swagger spec is available at:
```
http://localhost:8080/api/v1/swagger.yaml
```

### Option 2: Swagger UI (Online)
Visit [Swagger Editor](https://editor.swagger.io/) and paste the contents of `swagger.yaml`.

### Option 2: Local Swagger UI with Docker
```bash
docker run -p 8081:8080 -v $(pwd)/api:/usr/share/nginx/html/api -e SWAGGER_JSON=/usr/share/nginx/html/api/swagger.yaml swaggerapi/swagger-ui
```
Then open http://localhost:8081

### Option 3: Redoc (Alternative Viewer)
```bash
docker run -p 8081:80 -e SPEC_URL=https://raw.githubusercontent.com/your-repo/restic-monitor/main/api/swagger.yaml redocly/redoc
```

## API Overview

The Restic Monitor API provides endpoints for:

- **Status Monitoring**: Get current status of all backup targets
- **Snapshot Management**: View snapshots and their file contents
- **Repository Management**: Unlock locked repositories, prune old snapshots, toggle monitoring

## Authentication

The API supports two authentication methods:

1. **HTTP Basic Auth**: Username and password set via `AUTH_USERNAME` and `AUTH_PASSWORD` environment variables
2. **Bearer Token**: API token set via `AUTH_TOKEN` environment variable

Example with Basic Auth:
```bash
curl -u username:password http://localhost:8080/api/v1/status
```

Example with Bearer Token:
```bash
curl -H "Authorization: Bearer your-token-here" http://localhost:8080/api/v1/status
```

## Endpoints Summary

### GET /api/v1/status
Returns status of all configured backup targets.

### GET /api/v1/snapshots/{name}
Returns list of snapshots for a specific target.

### GET /api/v1/snapshot/{id}
Returns file list for a specific snapshot.

### POST /api/v1/unlock/{name}
Unlocks a locked repository.

### POST /api/v1/prune/{name}
Prunes old snapshots according to retention policy. Use `all` as the name to prune all targets.

### POST /api/v1/toggle/{name}
Toggles monitoring on/off for a target.

## Example Requests

### Get all backup statuses
```bash
curl http://localhost:8080/api/v1/status
```

### Get snapshots for "home" backup
```bash
curl http://localhost:8080/api/v1/snapshots/home
```

### Unlock a repository
```bash
curl -X POST http://localhost:8080/api/v1/unlock/home
```

### Prune old snapshots
```bash
curl -X POST http://localhost:8080/api/v1/prune/home
```

### Prune all repositories
```bash
curl -X POST http://localhost:8080/api/v1/prune/all
```

### Toggle monitoring
```bash
curl -X POST http://localhost:8080/api/v1/toggle/home
```

## Response Formats

All successful responses return JSON. Error responses may return plain text with HTTP status codes:

- `200 OK`: Successful operation
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or invalid
- `404 Not Found`: Target or resource not found
- `405 Method Not Allowed`: Wrong HTTP method used
- `500 Internal Server Error`: Server-side error

## CORS

The API includes CORS headers allowing cross-origin requests from any domain with the following allowed methods:
- GET
- POST
- OPTIONS

Allowed headers:
- Content-Type
- Authorization
