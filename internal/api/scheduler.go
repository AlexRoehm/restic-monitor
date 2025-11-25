package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/restic-monitor/internal/scheduler"
	"github.com/google/uuid"
)

// SchedulerStatusResponse represents the scheduler's current state
type SchedulerStatusResponse struct {
	Running          bool                      `json:"running"`
	LastRun          time.Time                 `json:"lastRun"`
	TotalRuns        int64                     `json:"totalRuns"`
	TasksGenerated   int64                     `json:"tasksGenerated"`
	ErrorsTotal      int64                     `json:"errorsTotal"`
	LastError        string                    `json:"lastError,omitempty"`
	PoliciesEnabled  int                       `json:"policiesEnabled"`
	UpcomingSchedule []UpcomingScheduleItem    `json:"upcomingSchedule"`
	Metrics          scheduler.MetricsSnapshot `json:"metrics"`
}

// UpcomingScheduleItem represents a scheduled task
type UpcomingScheduleItem struct {
	PolicyID   uuid.UUID `json:"policyId"`
	PolicyName string    `json:"policyName"`
	TaskType   string    `json:"taskType"`
	NextRun    time.Time `json:"nextRun"`
	Schedule   string    `json:"schedule"`
}

// handleSchedulerStatus godoc
// @Summary Get scheduler status and upcoming schedules
// @Description Returns the current state of the policy-based scheduler including metrics and upcoming task schedules
// @Tags Scheduler
// @Produce json
// @Success 200 {object} SchedulerStatusResponse
// @Failure 500 {object} ErrorResponse
// @Router /scheduler/status [get]
func (a *API) handleSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := r.Context()

	// Get scheduler metrics
	metrics := a.scheduler.GetMetrics()

	// Get all enabled policies
	policies, err := a.store.ListPolicies(ctx)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to list policies")
		return
	}

	// Count enabled policies and build upcoming schedule
	enabledCount := 0
	upcoming := []UpcomingScheduleItem{}

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		enabledCount++

		// Check if this policy has scheduled runs in metrics
		if taskTypes, ok := metrics.NextRunsByPolicy[policy.ID]; ok {
			// Add backup schedule
			if nextRun, ok := taskTypes["backup"]; ok {
				upcoming = append(upcoming, UpcomingScheduleItem{
					PolicyID:   policy.ID,
					PolicyName: policy.Name,
					TaskType:   "backup",
					NextRun:    nextRun,
					Schedule:   policy.Schedule,
				})
			}

			// Add check schedule if configured
			if policy.CheckSchedule != nil && *policy.CheckSchedule != "" {
				if nextRun, ok := taskTypes["check"]; ok {
					upcoming = append(upcoming, UpcomingScheduleItem{
						PolicyID:   policy.ID,
						PolicyName: policy.Name,
						TaskType:   "check",
						NextRun:    nextRun,
						Schedule:   *policy.CheckSchedule,
					})
				}
			}

			// Add prune schedule if configured
			if policy.PruneSchedule != nil && *policy.PruneSchedule != "" {
				if nextRun, ok := taskTypes["prune"]; ok {
					upcoming = append(upcoming, UpcomingScheduleItem{
						PolicyID:   policy.ID,
						PolicyName: policy.Name,
						TaskType:   "prune",
						NextRun:    nextRun,
						Schedule:   *policy.PruneSchedule,
					})
				}
			}
		}
	}

	response := SchedulerStatusResponse{
		Running:          a.scheduler.IsRunning(),
		LastRun:          metrics.LastRunTimestamp,
		TotalRuns:        metrics.TotalRuns,
		TasksGenerated:   metrics.TasksGeneratedTotal,
		ErrorsTotal:      metrics.ErrorsTotal,
		LastError:        metrics.LastError,
		PoliciesEnabled:  enabledCount,
		UpcomingSchedule: upcoming,
		Metrics:          metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
