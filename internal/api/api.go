package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/store"
)

// Run starts the HTTP API and shuts it down when the context is canceled.
func Run(ctx context.Context, addr string, cfg config.Config, st *store.Store) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: New(cfg, st).Handler(),
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Printf("listening on %s", addr)
	return srv.ListenAndServe()
}

// API exposes backup status endpoints.
type API struct {
	config config.Config
	store  *store.Store
}

// New constructs a new API handler.
func New(cfg config.Config, st *store.Store) *API {
	return &API{config: cfg, store: st}
}

// Handler registers routes.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", a.handleStatus)
	mux.HandleFunc("/unlock/", a.handleUnlock)
	return mux
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.URL.Query().Get("name")

	w.Header().Set("Content-Type", "application/json")

	if name != "" {
		status, err := a.store.GetStatus(ctx, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(statusPayload(status))
		return
	}

	statuses, err := a.store.ListStatuses(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payloads := make([]statusResponse, 0, len(statuses))
	for _, status := range statuses {
		payloads = append(payloads, statusPayload(status))
	}
	_ = json.NewEncoder(w).Encode(payloads)
}

func statusPayload(status store.BackupStatus) statusResponse {
	files := make([]fileResponse, 0, len(status.Files))
	for _, file := range status.Files {
		files = append(files, fileResponse{
			Path: file.Path,
			Name: file.Name,
			Type: file.Type,
			Size: file.Size,
		})
	}

	return statusResponse{
		Name:          status.Name,
		Repository:    status.Repository,
		LatestBackup:  status.LatestBackup,
		SnapshotCount: status.SnapshotCount,
		Health:        status.Health,
		StatusMessage: status.StatusMessage,
		CheckedAt:     status.CheckedAt,
		Files:         files,
	}
}

type statusResponse struct {
	Name          string         `json:"name"`
	Repository    string         `json:"repository"`
	LatestBackup  time.Time      `json:"latestBackup"`
	SnapshotCount int            `json:"snapshotCount"`
	Health        bool           `json:"health"`
	StatusMessage string         `json:"statusMessage"`
	CheckedAt     time.Time      `json:"checkedAt"`
	Files         []fileResponse `json:"files"`
}

type fileResponse struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

func (a *API) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	// Extract name from path: /unlock/{name}
	name := strings.TrimPrefix(r.URL.Path, "/unlock/")
	if name == "" {
		http.Error(w, "target name required", http.StatusBadRequest)
		return
	}

	// Get target from database
	targets, err := a.store.ListTargets(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("list targets: %v", err), http.StatusInternalServerError)
		return
	}

	var target *store.Target
	for _, t := range targets {
		if t.Name == name {
			target = &t
			break
		}
	}

	if target == nil {
		http.Error(w, fmt.Sprintf("target %s not found", name), http.StatusNotFound)
		return
	}

	// Run restic unlock
	log.Printf("unlocking repository for target %s", name)
	cmd := exec.CommandContext(ctx, a.config.ResticBinary, "unlock")
	cmd.Env = append(os.Environ(), a.envForTarget(*target)...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("unlock failed for %s: %v, output: %s", name, err, string(out))
		http.Error(w, fmt.Sprintf("unlock failed: %v\\n%s", err, string(out)), http.StatusInternalServerError)
		return
	}

	log.Printf("successfully unlocked repository for target %s", name)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "unlocked",
		"target":  name,
		"message": strings.TrimSpace(string(out)),
	})
}

func (a *API) envForTarget(target store.Target) []string {
	env := []string{fmt.Sprintf("RESTIC_REPOSITORY=%s", target.Repository)}
	if target.Password != "" {
		env = append(env, fmt.Sprintf("RESTIC_PASSWORD=%s", target.Password))
	}
	if target.PasswordFile != "" {
		env = append(env, fmt.Sprintf("RESTIC_PASSWORD_FILE=%s", target.PasswordFile))
	}
	cert := target.CertificateFile
	if cert == "" {
		cert = a.config.CertificateFile
	}
	if cert != "" {
		env = append(env, fmt.Sprintf("RESTIC_CACERT=%s", cert))
	}
	return env
}
