# Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.24-alpine AS backend-builder
WORKDIR /build
RUN apk add --no-cache git gcc musl-dev sqlite-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o restic-monitor cmd/restic-monitor/main.go

# Final image
FROM alpine:3.19
LABEL org.opencontainers.image.source=https://github.com/guxxde/restic-monitor
LABEL org.opencontainers.image.description="Restic Backup Monitor"
LABEL org.opencontainers.image.licenses=MIT

# Install restic and ca-certificates
RUN apk add --no-cache restic ca-certificates tzdata sqlite

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /build/restic-monitor .

# Copy frontend dist
COPY --from=frontend-builder /build/dist ./frontend/dist

# Create data, config, and public directories
RUN mkdir -p /app/data /app/config /app/public

# Create non-root user
RUN addgroup -g 1000 restic && \
    adduser -D -u 1000 -G restic restic && \
    chown -R restic:restic /app

USER restic

EXPOSE 8080

ENV DATABASE_DSN=/app/data/restic-monitor.db \
    TARGETS_FILE=/app/config/targets.json \
    API_LISTEN_ADDR=:8080 \
    CHECK_INTERVAL=5m \
    RESTIC_TIMEOUT=2m \
    SNAPSHOT_FILE_LIMIT=200 \
    STATIC_DIR=/app/frontend/dist \
    PUBLIC_DIR=/app/public

VOLUME ["/app/data", "/app/config", "/app/public"]

CMD ["./restic-monitor"]
