package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/config"
	"github.com/example/restic-monitor/internal/scheduler"
	"github.com/example/restic-monitor/internal/store"
)

// Run starts the HTTP API and shuts it down when the context is canceled.
func Run(ctx context.Context, addr string, cfg config.Config, st *store.Store, mon Monitor, staticDir string) error {
	api := New(cfg, st, mon, staticDir)
	
	// Start agent monitoring to mark stale agents offline
	api.StartAgentMonitor(ctx)
	
	srv := &http.Server{
		Addr:    addr,
		Handler: api.Handler(),
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Printf("listening on %s", addr)
	return srv.ListenAndServe()
}

// Monitor provides trigger mechanism for immediate checks
type Monitor interface {
	TriggerCheck(targetName string)
}

// Scheduler provides access to scheduler status and metrics
type Scheduler interface {
	IsRunning() bool
	GetMetrics() scheduler.MetricsSnapshot
}

// API exposes backup status endpoints.
type API struct {
	config    config.Config
	store     *store.Store
	monitor   Monitor
	scheduler Scheduler
	staticDir string
}

// New constructs a new API handler.
func New(cfg config.Config, st *store.Store, mon Monitor, staticDir string) *API {
	return &API{config: cfg, store: st, monitor: mon, scheduler: nil, staticDir: staticDir}
}

// NewWithScheduler constructs a new API handler with scheduler support.
func NewWithScheduler(cfg config.Config, st *store.Store, mon Monitor, sched Scheduler, staticDir string) *API {
	return &API{config: cfg, store: st, monitor: mon, scheduler: sched, staticDir: staticDir}
}

// Handler registers routes.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	// Agent API routes
	mux.HandleFunc("/agents/register", a.handleAgentRegister)
	mux.HandleFunc("/agents", a.handleGetAgents)     // GET /agents (list)

	// Policy API routes
	mux.HandleFunc("/policies/", a.handlePolicies) // Handles GET/PUT/DELETE for /policies/{id}
	mux.HandleFunc("/policies", a.handlePolicies)  // Handles POST/GET for /policies

	// Policy assignment routes
	assignmentHandler := NewPolicyAssignmentHandler(a.store.GetDB())
	mux.Handle("/agents/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a policy assignment route
		if strings.Contains(r.URL.Path, "/policies/") {
			// Add tenant ID header from store
			r.Header.Set("X-Tenant-ID", a.store.GetTenantID().String())
			assignmentHandler.ServeHTTP(w, r)
			return
		}
		// Check if this is a backup-runs route
		if strings.Contains(r.URL.Path, "/backup-runs") {
			if strings.Count(r.URL.Path, "/") == 4 {
				// /agents/{id}/backup-runs/{runId}
				a.handleGetBackupRun(w, r)
			} else {
				// /agents/{id}/backup-runs
				a.handleGetBackupRuns(w, r)
			}
			return
		}
		// Otherwise, route to existing agent handlers
		a.handleAgentsRouter(w, r)
	}))

	// Scheduler API routes
	if a.scheduler != nil {
		mux.HandleFunc("/scheduler/status", a.handleSchedulerStatus)
	}

	// API routes under /api/v1/
	mux.HandleFunc("/api/v1/status", a.handleStatus)
	mux.HandleFunc("/api/v1/status/", a.handleStatusByName)
	mux.HandleFunc("/api/v1/snapshots/", a.handleSnapshots)
	mux.HandleFunc("/api/v1/snapshot/", a.handleSnapshotFiles)
	mux.HandleFunc("/api/v1/unlock/", a.handleUnlock)
	mux.HandleFunc("/api/v1/prune/", a.handlePrune)
	mux.HandleFunc("/api/v1/toggle/", a.handleToggleDisabled)

	// Serve Swagger UI if enabled
	if a.config.ShowSwagger {
		mux.HandleFunc("/api/v1/swagger", a.handleSwagger)
		mux.HandleFunc("/api/v1/swagger.yaml", a.handleSwaggerSpec)
	}

	// Serve file lists from public directory
	if a.config.PublicDir != "" {
		publicFS := http.FileServer(http.Dir(a.config.PublicDir))
		mux.Handle("/api/v1/files/", http.StripPrefix("/api/v1/files/", publicFS))
	}

	// Serve static files from frontend/dist
	if a.staticDir != "" {
		fs := http.FileServer(http.Dir(a.staticDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For SPA routing: serve index.html for non-API routes
			if !strings.HasPrefix(r.URL.Path, "/api/") && !fileExists(a.staticDir, r.URL.Path) {
				http.ServeFile(w, r, a.staticDir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		}))
	}

	var handler http.Handler = mux

	// Wrap API routes with auth middleware if credentials are configured
	if (a.config.AuthUsername != "" && a.config.AuthPassword != "") || a.config.AuthToken != "" {
		handler = a.authMiddleware(handler)
	}

	// Wrap with CORS middleware (must be outermost)
	return a.corsMiddleware(handler)
}

// corsMiddleware adds CORS headers to all responses
func (a *API) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// fileExists checks if a file exists in the static directory
func fileExists(staticDir, path string) bool {
	if path == "/" {
		path = "/index.html"
	}
	fullPath := staticDir + path
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// authMiddleware wraps a handler with HTTP Basic Authentication and Bearer token support
// Only protects /api/ and /agents routes, excludes swagger endpoints
func (a *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only require auth for API routes and agent routes
		requiresAuth := strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/agents/") ||
			r.URL.Path == "/agents" ||
			strings.HasPrefix(r.URL.Path, "/policies/") ||
			r.URL.Path == "/policies"
		if !requiresAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Exclude Swagger UI and spec from authentication
		if strings.HasPrefix(r.URL.Path, "/api/v1/swagger") {
			next.ServeHTTP(w, r)
			return
		}

		// Check for Bearer token first
		if a.config.AuthToken != "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == a.config.AuthToken {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Fall back to Basic Auth if configured
		if a.config.AuthUsername != "" && a.config.AuthPassword != "" {
			username, password, ok := r.BasicAuth()
			if ok && username == a.config.AuthUsername && password == a.config.AuthPassword {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="Restic Monitor"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// handleStatus godoc
// @Summary Get status of all backup targets
// @Description Returns the current status of all configured backup targets including health, snapshot counts, and last check time
// @Tags Status
// @Accept json
// @Produce json
// @Param name query string false "Optional filter by target name"
// @Success 200 {object} statusResponse "Single status when name parameter is provided"
// @Success 200 {array} statusResponse "Array of statuses when no name parameter"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /status [get]
func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.URL.Query().Get("name")

	w.Header().Set("Content-Type", "application/json")

	// Get all targets to include disabled status
	targets, err := a.store.ListTargets(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("list targets: %v", err), http.StatusInternalServerError)
		return
	}

	targetMap := make(map[string]bool)
	for _, t := range targets {
		targetMap[t.Name] = t.Disabled
	}

	if name != "" {
		status, err := a.store.GetStatus(ctx, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(statusPayload(status, targetMap[status.Name], status.Health))
		return
	}

	statuses, err := a.store.ListStatuses(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payloads := make([]statusResponse, 0, len(statuses))
	for _, status := range statuses {
		payloads = append(payloads, statusPayload(status, targetMap[status.Name], status.Health))
	}
	_ = json.NewEncoder(w).Encode(payloads)
}

// handleStatusByName godoc
// @Summary Get status of a specific backup target
// @Description Returns the current status of a single backup target by name. Optionally check if the latest snapshot is younger than a specified age.
// @Tags Status
// @Accept json
// @Produce json
// @Param name path string true "Name of the backup target"
// @Param maxage query int false "Maximum age in hours for the latest snapshot. If specified, health status will be true only if repository is healthy AND snapshot is younger than maxage hours" minimum(1)
// @Success 200 {object} statusResponse "Successful response with backup status"
// @Failure 400 {string} string "Bad request - invalid maxage parameter"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /status/{name} [get]
func (a *API) handleStatusByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract name from path: /api/v1/status/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/status/")
	if name == "" {
		http.Error(w, "target name required", http.StatusBadRequest)
		return
	}

	// Parse maxage query parameter (in hours)
	var maxAgeHours int
	if maxAgeStr := r.URL.Query().Get("maxage"); maxAgeStr != "" {
		parsed, err := strconv.Atoi(maxAgeStr)
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid maxage parameter, must be positive integer", http.StatusBadRequest)
			return
		}
		maxAgeHours = parsed
	}

	// Get target to include disabled status
	targets, err := a.store.ListTargets(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("list targets: %v", err), http.StatusInternalServerError)
		return
	}

	var disabled bool
	for _, t := range targets {
		if t.Name == name {
			disabled = t.Disabled
			break
		}
	}

	status, err := a.store.GetStatus(ctx, name)
	if err != nil {
		http.Error(w, "target not found", http.StatusNotFound)
		return
	}

	// Check snapshot age if maxage is specified
	healthWithAge := status.Health
	if maxAgeHours > 0 {
		maxAge := time.Duration(maxAgeHours) * time.Hour
		age := time.Since(status.LatestBackup)
		healthWithAge = status.Health && age <= maxAge
	}

	_ = json.NewEncoder(w).Encode(statusPayload(status, disabled, healthWithAge))
}

func statusPayload(status store.BackupStatus, disabled bool, health bool) statusResponse {
	return statusResponse{
		Name:             status.Name,
		LatestBackup:     status.LatestBackup,
		LatestSnapshotID: status.LatestSnapshotID,
		SnapshotCount:    status.SnapshotCount,
		FileCount:        status.FileCount,
		Health:           health,
		StatusMessage:    status.StatusMessage,
		CheckedAt:        status.CheckedAt,
		Disabled:         disabled,
	}
}

type statusResponse struct {
	Name             string    `json:"name" example:"home"`
	LatestBackup     time.Time `json:"latestBackup" example:"2025-11-23T14:30:00Z"`
	LatestSnapshotID string    `json:"latestSnapshotID" example:"a1b2c3d4"`
	SnapshotCount    int       `json:"snapshotCount" example:"42"`
	FileCount        int       `json:"fileCount" example:"1234"`
	Health           bool      `json:"health" example:"true"`
	StatusMessage    string    `json:"statusMessage" example:"restic check succeeded"`
	CheckedAt        time.Time `json:"checkedAt" example:"2025-11-23T15:00:00Z"`
	Disabled         bool      `json:"disabled" example:"false"`
}

type snapshotResponse struct {
	ID       string    `json:"short_id" example:"a1b2c3d4"`
	Time     time.Time `json:"time" example:"2025-11-23T14:30:00Z"`
	Hostname string    `json:"hostname" example:"myserver"`
	Username string    `json:"username" example:"admin"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags"`
}

type fileResponse struct {
	Path string `json:"path" example:"/home/user/documents/file.txt"`
	Name string `json:"name" example:"file.txt"`
	Type string `json:"type" example:"file"`
	Size int64  `json:"size" example:"2048"`
}

// handleSnapshots godoc
// @Summary Get snapshots for a specific backup target
// @Description Returns list of all snapshots for the specified backup target
// @Tags Snapshots
// @Accept json
// @Produce json
// @Param name path string true "Name of the backup target"
// @Success 200 {array} snapshotResponse "List of snapshots"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /snapshots/{name} [get]
func (a *API) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")

	// Extract name from path: /api/v1/snapshots/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/snapshots/")
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

	// Execute restic snapshots command
	cmd := exec.CommandContext(ctx, a.config.ResticBinary, "snapshots", "--json", "--no-lock")
	cmd.Env = append(os.Environ(), a.envForTarget(*target)...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("snapshots failed for %s: %v, output: %s", name, err, string(out))
		http.Error(w, fmt.Sprintf("failed to get snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse and return snapshots
	var snapshots []map[string]interface{}
	if err := json.Unmarshal(out, &snapshots); err != nil {
		log.Printf("failed to parse snapshots JSON: %v", err)
		http.Error(w, fmt.Sprintf("failed to parse snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(snapshots)
}

// handleSnapshotFiles godoc
// @Summary Get file list for a specific snapshot
// @Description Returns the list of files contained in the specified snapshot
// @Tags Snapshots
// @Accept json
// @Produce json
// @Param id path string true "Snapshot ID"
// @Success 200 {array} fileResponse "List of files"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Snapshot file list not found"
// @Failure 500 {string} string "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /snapshot/{id} [get]
func (a *API) handleSnapshotFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract snapshot ID from path: /api/v1/snapshot/{id}
	snapshotID := strings.TrimPrefix(r.URL.Path, "/api/v1/snapshot/")
	if snapshotID == "" {
		http.Error(w, "snapshot ID required", http.StatusBadRequest)
		return
	}

	// Read file list from public directory
	filePath := fmt.Sprintf("%s/%s.txt", a.config.PublicDir, snapshotID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "snapshot files not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to read file list: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse JSONL and return as array
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	files := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var file map[string]interface{}
		if err := json.Unmarshal([]byte(line), &file); err != nil {
			continue
		}
		files = append(files, file)
	}

	_ = json.NewEncoder(w).Encode(files)
}

// handleUnlock godoc
// @Summary Unlock a repository
// @Description Removes stale locks from a Restic repository
// @Tags Maintenance
// @Accept json
// @Produce json
// @Param name path string true "Name of the backup target"
// @Success 200 {object} map[string]string "Repository unlocked successfully"
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Unlock failed"
// @Security BasicAuth
// @Security BearerAuth
// @Router /unlock/{name} [post]
func (a *API) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	// Extract name from path: /api/v1/unlock/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/unlock/")
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
	var unlockOutput string
	if a.config.MockMode {
		log.Printf("MOCK MODE - skipping restic unlock for target %s", name)
		unlockOutput = "mock unlock successful"
	} else {
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
		unlockOutput = strings.TrimSpace(string(out))
	}

	// Trigger immediate re-check of the repository
	a.monitor.TriggerCheck(name)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "unlocked",
		"target":  name,
		"message": unlockOutput,
	})
}

// handlePrune godoc
// @Summary Prune snapshots for a target or all targets
// @Description Applies retention policy (restic forget) to remove old snapshots. Use "all" as the name to prune all targets.
// @Tags Maintenance
// @Accept json
// @Produce json
// @Param name path string true "Name of the backup target or 'all' for all targets"
// @Success 200 {object} map[string]interface{} "Prune operation completed successfully"
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Prune operation failed"
// @Security BasicAuth
// @Security BearerAuth
// @Router /prune/{name} [post]
func (a *API) handlePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	// Extract name from path: /api/v1/prune/{name} or "all" for all targets
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/prune/")
	if name == "" {
		http.Error(w, "target name required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if name == "all" {
		// Prune all targets
		targets, err := a.store.ListTargets(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("list targets: %v", err), http.StatusInternalServerError)
			return
		}

		for _, target := range targets {
			if err := a.pruneTarget(ctx, target); err != nil {
				log.Printf("prune failed for %s: %v", target.Name, err)
			}
		}

		// Trigger re-check for all targets
		for _, target := range targets {
			a.monitor.TriggerCheck(target.Name)
		}

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "pruned all"})
		return
	}

	// Prune single target
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

	if err := a.pruneTarget(ctx, *target); err != nil {
		log.Printf("prune failed for %s: %v", name, err)
		http.Error(w, fmt.Sprintf("prune failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Trigger immediate re-check
	a.monitor.TriggerCheck(name)

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "pruned"})
}

func (a *API) pruneTarget(ctx context.Context, target store.Target) error {
	log.Printf("pruning target %s with policy: keep-last=%d keep-daily=%d keep-weekly=%d keep-monthly=%d",
		target.Name, target.KeepLast, target.KeepDaily, target.KeepWeekly, target.KeepMonthly)

	// Get snapshot IDs before pruning
	snapshotsBefore, err := a.getSnapshotIDs(ctx, target)
	if err != nil {
		log.Printf("failed to get snapshots before pruning %s: %v", target.Name, err)
		// Continue anyway - we'll just delete all file lists as fallback
	}

	// Execute restic forget with prune policy
	if a.config.MockMode {
		log.Printf("MOCK MODE - skipping restic forget for target %s", target.Name)
	} else {
		timeoutCtx, cancel := context.WithTimeout(ctx, a.config.ResticTimeout*3) // Prune can take longer
		defer cancel()

		args := []string{"forget", "--verbose"}

		// Add retention policy flags
		if target.KeepLast > 0 {
			args = append(args, "--keep-last", fmt.Sprintf("%d", target.KeepLast))
		}
		if target.KeepDaily > 0 {
			args = append(args, "--keep-daily", fmt.Sprintf("%d", target.KeepDaily))
		}
		if target.KeepWeekly > 0 {
			args = append(args, "--keep-weekly", fmt.Sprintf("%d", target.KeepWeekly))
		}
		if target.KeepMonthly > 0 {
			args = append(args, "--keep-monthly", fmt.Sprintf("%d", target.KeepMonthly))
		}

		cmd := exec.CommandContext(timeoutCtx, a.config.ResticBinary, args...)
		cmd.Env = append(os.Environ(), a.envForTarget(target)...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("forget failed for %s: %v, output: %s", target.Name, err, string(out))
			return fmt.Errorf("restic forget: %w", err)
		}

		log.Printf("forget completed for %s, output: %s", target.Name, string(out))
	}

	// Get snapshot IDs after pruning
	snapshotsAfter, err := a.getSnapshotIDs(ctx, target)
	if err != nil {
		log.Printf("failed to get snapshots after pruning %s: %v", target.Name, err)
		// Continue anyway - we'll just delete all file lists as fallback
	}

	// Determine which snapshots were removed
	var removedSnapshots []string
	if len(snapshotsBefore) > 0 && len(snapshotsAfter) > 0 {
		afterSet := make(map[string]bool)
		for _, id := range snapshotsAfter {
			afterSet[id] = true
		}
		for _, id := range snapshotsBefore {
			if !afterSet[id] {
				removedSnapshots = append(removedSnapshots, id)
			}
		}
	}

	// Delete file lists from public directory
	if a.config.PublicDir != "" {
		if len(removedSnapshots) > 0 {
			// Delete only the file lists for removed snapshots
			for _, snapshotID := range removedSnapshots {
				filePath := filepath.Join(a.config.PublicDir, snapshotID+".txt")
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					log.Printf("failed to delete file list %s: %v", filePath, err)
				} else if err == nil {
					log.Printf("deleted file list for removed snapshot %s", snapshotID)
				}
			}
			log.Printf("deleted %d file lists for target %s", len(removedSnapshots), target.Name)
		} else {
			// Fallback: if we couldn't determine which snapshots were removed, delete all
			log.Printf("couldn't determine removed snapshots, deleting all file lists for safety")
			pattern := fmt.Sprintf("%s/*.txt", a.config.PublicDir)
			files, err := filepath.Glob(pattern)
			if err == nil {
				for _, file := range files {
					if err := os.Remove(file); err != nil {
						log.Printf("failed to delete file list %s: %v", file, err)
					}
				}
				log.Printf("deleted all file lists for target %s", target.Name)
			}
		}
	}

	return nil
}

// getSnapshotIDs returns all snapshot IDs for a target
func (a *API) getSnapshotIDs(ctx context.Context, target store.Target) ([]string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.ResticTimeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, a.config.ResticBinary, "snapshots", "--json", "--no-lock")
	cmd.Env = append(os.Environ(), a.envForTarget(target)...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("restic snapshots: %w", err)
	}

	var snapshots []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &snapshots); err != nil {
		return nil, fmt.Errorf("unmarshal snapshots: %w", err)
	}

	ids := make([]string, len(snapshots))
	for i, s := range snapshots {
		ids[i] = s.ID
	}

	return ids, nil
}

// handleToggleDisabled godoc
// @Summary Enable or disable monitoring for a target
// @Description Toggles the disabled state of a backup target
// @Tags Configuration
// @Accept json
// @Produce json
// @Param name path string true "Name of the backup target"
// @Success 200 {object} map[string]interface{} "Target state toggled successfully"
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Target not found"
// @Failure 500 {string} string "Internal server error"
// @Security BasicAuth
// @Security BearerAuth
// @Router /toggle/{name} [post]
func (a *API) handleToggleDisabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	// Extract name from path: /api/v1/toggle/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/toggle/")
	if name == "" {
		http.Error(w, "target name required", http.StatusBadRequest)
		return
	}

	if err := a.store.ToggleTargetDisabled(ctx, name); err != nil {
		log.Printf("toggle disabled failed for %s: %v", name, err)
		http.Error(w, fmt.Sprintf("toggle failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("toggled disabled state for target %s", name)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "toggled"})
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

func (a *API) handleSwaggerSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, "api/swagger.yaml")
}

func (a *API) handleSwagger(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Restic Monitor API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/api/v1/swagger.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                persistAuthorization: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
