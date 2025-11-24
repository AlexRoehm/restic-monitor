package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the settings needed by the restic monitor.
type Config struct {
	ResticBinary            string
	Repository              string
	Password                string
	PasswordFile            string
	CertificateFile         string
	CheckInterval           time.Duration
	ResticTimeout           time.Duration
	DatabaseDSN             string
	APIListenAddr           string
	SnapshotLimit           int
	TargetsFile             string
	AuthUsername            string
	AuthPassword            string
	AuthToken               string
	PublicDir               string
	ShowSwagger             bool
	MockMode                bool
	HeartbeatTimeoutSeconds int // Seconds before agent is considered offline
}

// Load reads configuration values from environment variables.
func Load() (Config, error) {
	cfg := Config{
		ResticBinary:            firstNonEmpty(os.Getenv("RESTIC_BINARY"), "restic"),
		Repository:              os.Getenv("RESTIC_REPOSITORY"),
		Password:                os.Getenv("RESTIC_PASSWORD"),
		PasswordFile:            os.Getenv("RESTIC_PASSWORD_FILE"),
		CertificateFile:         os.Getenv("RESTIC_CERT_FILE"),
		DatabaseDSN:             firstNonEmpty(os.Getenv("DATABASE_DSN"), "restic-monitor.db"),
		APIListenAddr:           firstNonEmpty(os.Getenv("API_LISTEN_ADDR"), ":8080"),
		CheckInterval:           mustParseDuration(os.Getenv("CHECK_INTERVAL"), 5*time.Minute),
		ResticTimeout:           mustParseDuration(os.Getenv("RESTIC_TIMEOUT"), 60*time.Second),
		SnapshotLimit:           mustParseInt(os.Getenv("SNAPSHOT_FILE_LIMIT"), 200),
		TargetsFile:             firstNonEmpty(os.Getenv("TARGETS_FILE"), "targets.json"),
		AuthUsername:            os.Getenv("AUTH_USERNAME"),
		AuthPassword:            os.Getenv("AUTH_PASSWORD"),
		AuthToken:               os.Getenv("AUTH_TOKEN"),
		PublicDir:               firstNonEmpty(os.Getenv("PUBLIC_DIR"), "public"),
		ShowSwagger:             os.Getenv("SHOW_SWAGGER") == "true",
		MockMode:                os.Getenv("MOCK_MODE") == "true",
		HeartbeatTimeoutSeconds: mustParseInt(os.Getenv("HEARTBEAT_TIMEOUT_SECONDS"), 90),
	}

	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mustParseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func mustParseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
