package store_test

import (
	"testing"
	"time"

	"github.com/example/restic-monitor/internal/store"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTaskModel tests the Task model (TDD - Epic 10.2)
func TestTaskModel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = store.MigrateModels(db)
	if err != nil {
		t.Fatalf("Failed to migrate models: %v", err)
	}

	task := &store.Task{
		TenantID:   uuid.New(),
		AgentID:    uuid.New(),
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "pending",
		Repository: "s3:bucket/repo",
	}

	result := db.Create(task)
	if result.Error != nil {
		t.Fatalf("Failed to create task: %v", result.Error)
	}

	if task.ID == uuid.Nil {
		t.Error("Task ID should be auto-generated")
	}

	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt should be auto-set")
	}
}

// TestTaskWithOptionalFields tests task with all fields (TDD - Epic 10.2)
func TestTaskWithOptionalFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = store.MigrateModels(db)
	if err != nil {
		t.Fatalf("Failed to migrate models: %v", err)
	}

	now := time.Now()
	task := &store.Task{
		TenantID:   uuid.New(),
		AgentID:    uuid.New(),
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "pending",
		Repository: "s3:bucket/repo",
		IncludePaths: store.JSONB{
			"paths": []interface{}{"/home", "/etc"},
		},
		ExcludePaths: store.JSONB{
			"paths": []interface{}{"*.tmp", "/cache"},
		},
		Retention: store.JSONB{
			"keepLast":  7,
			"keepDaily": 14,
		},
		ExecutionParams: store.JSONB{
			"parallelism":        4,
			"bandwidthLimitKbps": 5000,
		},
		ScheduledFor: &now,
	}

	result := db.Create(task)
	if result.Error != nil {
		t.Fatalf("Failed to create task: %v", result.Error)
	}

	// Retrieve task
	var retrieved store.Task
	db.First(&retrieved, task.ID)

	if retrieved.IncludePaths == nil {
		t.Error("IncludePaths should be preserved")
	}

	if retrieved.ExcludePaths == nil {
		t.Error("ExcludePaths should be preserved")
	}

	if retrieved.Retention == nil {
		t.Error("Retention should be preserved")
	}

	if retrieved.ExecutionParams == nil {
		t.Error("ExecutionParams should be preserved")
	}
}

// TestTaskStateTransitions tests task status changes (TDD - Epic 10.3)
func TestTaskStateTransitions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = store.MigrateModels(db)
	if err != nil {
		t.Fatalf("Failed to migrate models: %v", err)
	}

	task := &store.Task{
		TenantID:   uuid.New(),
		AgentID:    uuid.New(),
		PolicyID:   uuid.New(),
		TaskType:   "backup",
		Status:     "pending",
		Repository: "s3:bucket/repo",
	}

	db.Create(task)

	// Transition to assigned
	now := time.Now()
	task.Status = "assigned"
	task.AssignedAt = &now
	db.Save(task)

	var retrieved store.Task
	db.First(&retrieved, task.ID)

	if retrieved.Status != "assigned" {
		t.Errorf("Expected status 'assigned', got %s", retrieved.Status)
	}

	if retrieved.AssignedAt == nil {
		t.Error("AssignedAt should be set")
	}

	// Transition to in-progress
	now2 := time.Now()
	task.Status = "in-progress"
	task.AcknowledgedAt = &now2
	task.StartedAt = &now2
	db.Save(task)

	db.First(&retrieved, task.ID)

	if retrieved.Status != "in-progress" {
		t.Errorf("Expected status 'in-progress', got %s", retrieved.Status)
	}

	// Transition to completed
	now3 := time.Now()
	task.Status = "completed"
	task.CompletedAt = &now3
	db.Save(task)

	db.First(&retrieved, task.ID)

	if retrieved.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", retrieved.Status)
	}

	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

// TestTaskQuery tests querying tasks by agent and status (TDD - Epic 10.2)
func TestTaskQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = store.MigrateModels(db)
	if err != nil {
		t.Fatalf("Failed to migrate models: %v", err)
	}

	agentID := uuid.New()
	otherAgentID := uuid.New()

	// Create tasks
	tasks := []*store.Task{
		{
			TenantID:   uuid.New(),
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "backup",
			Status:     "pending",
			Repository: "s3:bucket/repo1",
		},
		{
			TenantID:   uuid.New(),
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "check",
			Status:     "pending",
			Repository: "s3:bucket/repo2",
		},
		{
			TenantID:   uuid.New(),
			AgentID:    agentID,
			PolicyID:   uuid.New(),
			TaskType:   "prune",
			Status:     "completed",
			Repository: "s3:bucket/repo3",
		},
		{
			TenantID:   uuid.New(),
			AgentID:    otherAgentID,
			PolicyID:   uuid.New(),
			TaskType:   "backup",
			Status:     "pending",
			Repository: "s3:bucket/repo4",
		},
	}

	for _, task := range tasks {
		db.Create(task)
	}

	// Query pending tasks for agentID
	var pendingTasks []store.Task
	db.Where("agent_id = ? AND status = ?", agentID, "pending").Find(&pendingTasks)

	if len(pendingTasks) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(pendingTasks))
	}

	// Query all tasks for agentID
	var allAgentTasks []store.Task
	db.Where("agent_id = ?", agentID).Find(&allAgentTasks)

	if len(allAgentTasks) != 3 {
		t.Errorf("Expected 3 tasks for agent, got %d", len(allAgentTasks))
	}

	// Query tasks for other agent
	var otherAgentTasks []store.Task
	db.Where("agent_id = ?", otherAgentID).Find(&otherAgentTasks)

	if len(otherAgentTasks) != 1 {
		t.Errorf("Expected 1 task for other agent, got %d", len(otherAgentTasks))
	}
}
