package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskResponse represents a task with calculated retry metadata
type TaskResponse struct {
	store.Task
	RetriesRemaining *int `json:"retries_remaining,omitempty"`
}

// buildTaskResponse creates a TaskResponse with calculated fields
func buildTaskResponse(task store.Task) TaskResponse {
	resp := TaskResponse{
		Task: task,
	}
	
	// Calculate retries remaining
	if task.RetryCount != nil && task.MaxRetries != nil {
		remaining := *task.MaxRetries - *task.RetryCount
		if remaining < 0 {
			remaining = 0
		}
		resp.RetriesRemaining = &remaining
	}
	
	return resp
}

// handleGetAgentTasks retrieves pending tasks for an agent and assigns them
// GET /agents/{agentId}/tasks?limit=10
func (a *API) handleGetAgentTasks(w http.ResponseWriter, r *http.Request) {
	// Extract agent ID from path
	agentIDStr := strings.TrimPrefix(r.URL.Path, "/agents/")
	agentIDStr = strings.TrimSuffix(agentIDStr, "/tasks")

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		log.Printf("[TASKS] Invalid agent ID format: %s", agentIDStr)
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Parse limit query parameter (default: 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Retrieve pending tasks and mark as assigned
	var tasks []store.Task
	now := time.Now()

	err = a.store.GetDB().Transaction(func(tx *gorm.DB) error {
		// Find pending tasks for this agent, ordered by scheduled_for
		result := tx.Where("agent_id = ? AND status = ?", agentID, "pending").
			Order("scheduled_for ASC, created_at ASC").
			Limit(limit).
			Find(&tasks)

		if result.Error != nil {
			return result.Error
		}

		// Update all retrieved tasks to 'assigned' status
		for i := range tasks {
			tasks[i].Status = "assigned"
			tasks[i].AssignedAt = &now
			if err := tx.Save(&tasks[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("[TASKS] Failed to retrieve tasks for agent %s: %v", agentID, err)
		http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
		return
	}

	log.Printf("[TASKS] Assigned %d task(s) to agent %s", len(tasks), agentID)

	// Build response with retry metadata
	taskResponses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = buildTaskResponse(task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": taskResponses,
	})
}

// handleAcknowledgeTask acknowledges task receipt and starts execution
// POST /agents/{agentId}/tasks/{taskId}/ack
func (a *API) handleAcknowledgeTask(w http.ResponseWriter, r *http.Request) {
	// Extract agent ID and task ID from path
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	agentIDStr := parts[0]
	taskIDStr := parts[2]

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		log.Printf("[TASKS] Invalid agent ID format in ack: %s", agentIDStr)
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		log.Printf("[TASKS] Invalid task ID format in ack: %s", taskIDStr)
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Update task status to in-progress
	now := time.Now()
	result := a.store.GetDB().Model(&store.Task{}).
		Where("id = ? AND agent_id = ?", taskID, agentID).
		Updates(map[string]interface{}{
			"status":          "in-progress",
			"acknowledged_at": now,
			"started_at":      now,
		})

	if result.Error != nil {
		log.Printf("[TASKS] Failed to acknowledge task %s for agent %s: %v", taskID, agentID, result.Error)
		http.Error(w, "Failed to acknowledge task", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("[TASKS] Task not found or unauthorized: task=%s, agent=%s", taskID, agentID)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	log.Printf("[TASKS] Task acknowledged: task=%s, agent=%s, status=in-progress", taskID, agentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "acknowledged",
		"message": "Task acknowledged and started",
	})
}
