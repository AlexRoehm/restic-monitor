package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
)

type Monitor struct {
	cfg   config.Config
	store *store.Store
}

func New(cfg config.Config, str *store.Store) *Monitor {
	return &Monitor{cfg: cfg, store: str}
}

func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.CheckInterval)
	defer ticker.Stop()

	log.Printf("monitor starting, running initial check")
	m.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Printf("monitor loop stopping")
			return
		case <-ticker.C:
			m.runOnce(ctx)
		}
	}
}

func (m *Monitor) runOnce(ctx context.Context) {
	targets, err := m.store.ListTargets(ctx)
	if err != nil {
		log.Printf("list restic targets: %v", err)
		return
	}
	if len(targets) == 0 {
		log.Printf("no restic targets configured")
		return
	}

	log.Printf("checking %d target(s)", len(targets))
	for _, target := range targets {
		m.runTarget(ctx, target)
	}
}

func (m *Monitor) runTarget(ctx context.Context, target store.Target) {
	log.Printf("checking target %s (repo: %s)", target.Name, target.Repository)
	data := store.StatusData{
		Name:       target.Name,
		Repository: target.Repository,
		CheckedAt:  time.Now(),
	}

	snapshots, err := m.listSnapshots(ctx, target)
	if err != nil {
		msg := fmt.Sprintf("list snapshots: %v", err)
		data.Health = false
		data.StatusMessage = joinStatus(data.StatusMessage, msg)
		_ = m.store.SaveStatus(ctx, data)
		log.Printf("target %s snapshots error: %v", target.Name, err)
		return
	}

	log.Printf("target %s: found %d snapshot(s)", target.Name, len(snapshots))
	data.SnapshotCount = len(snapshots)
	if latest := latestSnapshot(snapshots); latest != nil {
		data.LatestBackup = latest.Time
		log.Printf("target %s: latest snapshot %s from %s", target.Name, latest.ID, latest.Time.Format(time.RFC3339))
		files, err := m.listSnapshotFiles(ctx, target, latest.ID)
		if err != nil {
			log.Printf("target %s: error listing files: %v", target.Name, err)
			// Don't add to status message, just use empty files
		} else {
			log.Printf("target %s: retrieved %d file(s)", target.Name, len(files))
			data.Files = files
		}
	}

	log.Printf("target %s: running health check", target.Name)
	healthy, msg := m.checkHealth(ctx, target)
	// If repository is locked, treat as healthy but note the lock
	if strings.Contains(msg, "repository is already locked") {
		log.Printf("target %s: repository locked, skipping health check", target.Name)
		data.Health = true
		data.StatusMessage = joinStatus(data.StatusMessage, "repository locked (health check skipped)")
	} else {
		data.Health = healthy
		if msg != "" && !healthy {
			data.StatusMessage = joinStatus(data.StatusMessage, msg)
		}
	}
	log.Printf("target %s: health=%v", target.Name, data.Health)

	if err := m.store.SaveStatus(ctx, data); err != nil {
		log.Printf("persist status for %s: %v", target.Name, err)
	} else {
		log.Printf("target %s: status saved successfully", target.Name)
	}
}

func joinStatus(previous, addition string) string {
	if previous == "" {
		return addition
	}
	return fmt.Sprintf("%s | %s", previous, addition)
}

func (m *Monitor) listSnapshots(ctx context.Context, target store.Target) ([]resticSnapshot, error) {
	// Validate certificate file if specified
	certFile := target.CertificateFile
	if certFile == "" {
		certFile = m.cfg.CertificateFile
	}
	if certFile != "" {
		if _, err := os.Stat(certFile); err != nil {
			log.Printf("target %s: certificate file %s not accessible: %v", target.Name, certFile, err)
			return nil, fmt.Errorf("certificate file %s: %w", certFile, err)
		}
		log.Printf("target %s: using certificate file: %s", target.Name, certFile)
	}

	// Create timeout context (30 seconds for snapshots)
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, m.cfg.ResticBinary, "snapshots", "--json")
	cmd.Env = append(os.Environ(), m.envForTarget(target)...)

	log.Printf("target %s: executing: %s snapshots --json (timeout: 30s)", target.Name, m.cfg.ResticBinary)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Printf("target %s: restic snapshots timed out after 30s", target.Name)
			return nil, fmt.Errorf("restic snapshots: timeout")
		}
		log.Printf("target %s: restic snapshots failed: %v, output: %s", target.Name, err, string(out))
		return nil, fmt.Errorf("restic snapshots: %w", err)
	}

	log.Printf("target %s: restic snapshots output length: %d bytes", target.Name, len(out))

	var snapshots []resticSnapshot
	if err := json.Unmarshal(out, &snapshots); err != nil {
		log.Printf("target %s: failed to parse snapshots JSON: %v", target.Name, err)
		return nil, fmt.Errorf("parse snapshots: %w", err)
	}

	return snapshots, nil
}

func (m *Monitor) listSnapshotFiles(ctx context.Context, target store.Target, snapshotID string) ([]store.SnapshotFileData, error) {
	log.Printf("target %s: executing: %s ls %s --json", target.Name, m.cfg.ResticBinary, snapshotID)

	// Create timeout context (60 seconds for ls)
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, m.cfg.ResticBinary, "ls", snapshotID, "--json")
	cmd.Env = append(os.Environ(), m.envForTarget(target)...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	files := make([]store.SnapshotFileData, 0, m.cfg.SnapshotLimit)

	for scanner.Scan() {
		if len(files) >= m.cfg.SnapshotLimit {
			break
		}
		var entry resticLsEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		files = append(files, store.SnapshotFileData{
			Path: entry.Path,
			Name: entry.Name,
			Type: entry.Type,
			Size: entry.Size,
		})
	}

	if err := scanner.Err(); err != nil {
		log.Printf("target %s: scanner error reading ls output: %v", target.Name, err)
		return files, err
	}

	if err := cmd.Wait(); err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Printf("target %s: restic ls timed out after 60s, returning partial results (%d files)", target.Name, len(files))
			return files, nil // Return partial results on timeout
		}
		log.Printf("target %s: restic ls failed: %v", target.Name, err)
		return files, err
	}

	log.Printf("target %s: successfully listed %d files", target.Name, len(files))
	return files, nil
}

func (m *Monitor) checkHealth(ctx context.Context, target store.Target) (bool, string) {
	log.Printf("target %s: executing: %s check --json", target.Name, m.cfg.ResticBinary)

	// Create timeout context (60 seconds for check)
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, m.cfg.ResticBinary, "check", "--json")
	cmd.Env = append(os.Environ(), m.envForTarget(target)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Printf("target %s: restic check timed out after 60s", target.Name)
			return false, "restic check: timeout"
		}
		log.Printf("target %s: restic check failed: %v, output: %s", target.Name, err, strings.TrimSpace(string(out)))
		return false, fmt.Sprintf("restic check failed: %v %s", err, strings.TrimSpace(string(out)))
	}
	log.Printf("target %s: restic check succeeded", target.Name)
	return true, strings.TrimSpace(string(out))
}

func (m *Monitor) envForTarget(target store.Target) []string {
	env := []string{fmt.Sprintf("RESTIC_REPOSITORY=%s", target.Repository)}
	if target.Password != "" {
		env = append(env, fmt.Sprintf("RESTIC_PASSWORD=%s", "***")) // log as masked
	}
	if target.PasswordFile != "" {
		env = append(env, fmt.Sprintf("RESTIC_PASSWORD_FILE=%s", target.PasswordFile))
	}
	cert := target.CertificateFile
	if cert == "" {
		cert = m.cfg.CertificateFile
	}
	if cert != "" {
		env = append(env, fmt.Sprintf("RESTIC_CACERT=%s", cert))
	}
	log.Printf("target %s: env vars: %v", target.Name, env)

	// Build actual env with real password
	actualEnv := []string{fmt.Sprintf("RESTIC_REPOSITORY=%s", target.Repository)}
	if target.Password != "" {
		actualEnv = append(actualEnv, fmt.Sprintf("RESTIC_PASSWORD=%s", target.Password))
	}
	if target.PasswordFile != "" {
		actualEnv = append(actualEnv, fmt.Sprintf("RESTIC_PASSWORD_FILE=%s", target.PasswordFile))
	}
	if cert != "" {
		actualEnv = append(actualEnv, fmt.Sprintf("RESTIC_CACERT=%s", cert))
	}
	return actualEnv
}

func latestSnapshot(list []resticSnapshot) *resticSnapshot {
	if len(list) == 0 {
		return nil
	}
	latest := list[0]
	for i := 1; i < len(list); i++ {
		if list[i].Time.After(latest.Time) {
			latest = list[i]
		}
	}
	return &latest
}

type resticSnapshot struct {
	ID   string    `json:"short_id"`
	Time time.Time `json:"time"`
}

type resticLsEntry struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}
