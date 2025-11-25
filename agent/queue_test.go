package agent_test

import (
	"testing"
	"time"

	"github.com/example/restic-monitor/agent"
)

// TestTaskQueueEnqueue tests adding tasks to queue (TDD - Epic 9.5)
func TestTaskQueueEnqueue(t *testing.T) {
	queue := agent.NewTaskQueue()

	task := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("Expected enqueue to succeed, got error: %v", err)
	}

	if queue.Size() != 1 {
		t.Errorf("Expected queue size 1, got %d", queue.Size())
	}
}

// TestTaskQueueDuplicateDetection tests duplicate task detection (TDD - Epic 9.5)
func TestTaskQueueDuplicateDetection(t *testing.T) {
	queue := agent.NewTaskQueue()

	task := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	// First enqueue should succeed
	err := queue.Enqueue(task)
	if err != nil {
		t.Fatalf("First enqueue should succeed, got error: %v", err)
	}

	// Second enqueue of same task should fail
	err = queue.Enqueue(task)
	if err == nil {
		t.Fatal("Expected error for duplicate task")
	}

	// Queue should still have only 1 task
	if queue.Size() != 1 {
		t.Errorf("Expected queue size 1, got %d", queue.Size())
	}
}

// TestTaskQueueDequeue tests removing tasks from queue (TDD - Epic 9.5)
func TestTaskQueueDequeue(t *testing.T) {
	queue := agent.NewTaskQueue()

	task1 := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo1",
		CreatedAt:  time.Now(),
	}

	task2 := agent.Task{
		TaskID:     "223e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
		TaskType:   "check",
		Repository: "s3:bucket/repo2",
		CreatedAt:  time.Now(),
	}

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	// Dequeue first task
	dequeued := queue.Dequeue()
	if dequeued == nil {
		t.Fatal("Expected task, got nil")
	}
	if dequeued.TaskID != task1.TaskID {
		t.Errorf("Expected first task, got %s", dequeued.TaskID)
	}

	// Queue should have 1 task left
	if queue.Size() != 1 {
		t.Errorf("Expected queue size 1, got %d", queue.Size())
	}

	// Dequeue second task
	dequeued = queue.Dequeue()
	if dequeued == nil {
		t.Fatal("Expected task, got nil")
	}
	if dequeued.TaskID != task2.TaskID {
		t.Errorf("Expected second task, got %s", dequeued.TaskID)
	}

	// Queue should be empty
	if !queue.IsEmpty() {
		t.Error("Expected empty queue")
	}
}

// TestTaskQueueDequeueEmpty tests dequeuing from empty queue (TDD - Epic 9.5)
func TestTaskQueueDequeueEmpty(t *testing.T) {
	queue := agent.NewTaskQueue()

	dequeued := queue.Dequeue()
	if dequeued != nil {
		t.Error("Expected nil for empty queue")
	}
}

// TestTaskQueuePeek tests peeking at next task (TDD - Epic 9.5)
func TestTaskQueuePeek(t *testing.T) {
	queue := agent.NewTaskQueue()

	task := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	queue.Enqueue(task)

	// Peek should return task without removing
	peeked := queue.Peek()
	if peeked == nil {
		t.Fatal("Expected task, got nil")
	}
	if peeked.TaskID != task.TaskID {
		t.Errorf("Expected task %s, got %s", task.TaskID, peeked.TaskID)
	}

	// Queue size should not change
	if queue.Size() != 1 {
		t.Errorf("Expected queue size 1, got %d", queue.Size())
	}

	// Peek again should return same task
	peeked2 := queue.Peek()
	if peeked2 == nil || peeked2.TaskID != task.TaskID {
		t.Error("Second peek should return same task")
	}
}

// TestTaskQueueContains tests checking for task existence (TDD - Epic 9.5)
func TestTaskQueueContains(t *testing.T) {
	queue := agent.NewTaskQueue()

	task := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo",
		CreatedAt:  time.Now(),
	}

	// Should not contain before enqueue
	if queue.Contains(task.TaskID) {
		t.Error("Queue should not contain task before enqueue")
	}

	queue.Enqueue(task)

	// Should contain after enqueue
	if !queue.Contains(task.TaskID) {
		t.Error("Queue should contain task after enqueue")
	}

	queue.Dequeue()

	// Should not contain after dequeue
	if queue.Contains(task.TaskID) {
		t.Error("Queue should not contain task after dequeue")
	}
}

// TestTaskQueueClear tests clearing the queue (TDD - Epic 9.5)
func TestTaskQueueClear(t *testing.T) {
	queue := agent.NewTaskQueue()

	// Add multiple tasks
	for i := 0; i < 5; i++ {
		task := agent.Task{
			TaskID:     "123e4567-e89b-12d3-a456-42661417400" + string(rune('0'+i)),
			PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
			TaskType:   "backup",
			Repository: "s3:bucket/repo",
			CreatedAt:  time.Now(),
		}
		queue.Enqueue(task)
	}

	if queue.Size() != 5 {
		t.Errorf("Expected queue size 5, got %d", queue.Size())
	}

	queue.Clear()

	if !queue.IsEmpty() {
		t.Error("Queue should be empty after clear")
	}
	if queue.Size() != 0 {
		t.Errorf("Expected queue size 0, got %d", queue.Size())
	}
}

// TestTaskQueueGetAll tests getting all tasks (TDD - Epic 9.5)
func TestTaskQueueGetAll(t *testing.T) {
	queue := agent.NewTaskQueue()

	tasks := []agent.Task{
		{
			TaskID:     "123e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
			TaskType:   "backup",
			Repository: "s3:bucket/repo1",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     "223e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
			TaskType:   "check",
			Repository: "s3:bucket/repo2",
			CreatedAt:  time.Now(),
		},
	}

	for _, task := range tasks {
		queue.Enqueue(task)
	}

	allTasks := queue.GetAll()

	if len(allTasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(allTasks))
	}

	// Verify order is preserved
	for i, task := range allTasks {
		if task.TaskID != tasks[i].TaskID {
			t.Errorf("Task %d: expected %s, got %s", i, tasks[i].TaskID, task.TaskID)
		}
	}

	// Modifying returned slice should not affect queue
	allTasks[0].TaskType = "prune"
	if queue.Peek().TaskType != "backup" {
		t.Error("Modifying GetAll result should not affect queue")
	}
}

// TestTaskQueueRemove tests removing specific task (TDD - Epic 9.5)
func TestTaskQueueRemove(t *testing.T) {
	queue := agent.NewTaskQueue()

	task1 := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo1",
		CreatedAt:  time.Now(),
	}

	task2 := agent.Task{
		TaskID:     "223e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
		TaskType:   "check",
		Repository: "s3:bucket/repo2",
		CreatedAt:  time.Now(),
	}

	task3 := agent.Task{
		TaskID:     "323e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "434e5678-e89b-12d3-a456-426614174001",
		TaskType:   "prune",
		Repository: "s3:bucket/repo3",
		CreatedAt:  time.Now(),
	}

	queue.Enqueue(task1)
	queue.Enqueue(task2)
	queue.Enqueue(task3)

	// Remove middle task
	removed := queue.Remove(task2.TaskID)
	if !removed {
		t.Error("Expected task to be removed")
	}

	if queue.Size() != 2 {
		t.Errorf("Expected queue size 2, got %d", queue.Size())
	}

	if queue.Contains(task2.TaskID) {
		t.Error("Queue should not contain removed task")
	}

	// Try to remove non-existent task
	removed = queue.Remove("non-existent-id")
	if removed {
		t.Error("Should not remove non-existent task")
	}
}

// TestTaskQueueEnqueueMultiple tests adding multiple tasks (TDD - Epic 9.5)
func TestTaskQueueEnqueueMultiple(t *testing.T) {
	queue := agent.NewTaskQueue()

	tasks := []agent.Task{
		{
			TaskID:     "123e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
			TaskType:   "backup",
			Repository: "s3:bucket/repo1",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     "223e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
			TaskType:   "check",
			Repository: "s3:bucket/repo2",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     "323e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "434e5678-e89b-12d3-a456-426614174001",
			TaskType:   "prune",
			Repository: "s3:bucket/repo3",
			CreatedAt:  time.Now(),
		},
	}

	added, err := queue.EnqueueMultiple(tasks)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if added != 3 {
		t.Errorf("Expected 3 tasks added, got %d", added)
	}

	if queue.Size() != 3 {
		t.Errorf("Expected queue size 3, got %d", queue.Size())
	}
}

// TestTaskQueueEnqueueMultipleWithDuplicates tests adding multiple tasks with duplicates (TDD - Epic 9.5)
func TestTaskQueueEnqueueMultipleWithDuplicates(t *testing.T) {
	queue := agent.NewTaskQueue()

	task1 := agent.Task{
		TaskID:     "123e4567-e89b-12d3-a456-426614174000",
		PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
		TaskType:   "backup",
		Repository: "s3:bucket/repo1",
		CreatedAt:  time.Now(),
	}

	// Add first task
	queue.Enqueue(task1)

	// Try to add multiple tasks including duplicate
	tasks := []agent.Task{
		task1, // Duplicate
		{
			TaskID:     "223e4567-e89b-12d3-a456-426614174000",
			PolicyID:   "334e5678-e89b-12d3-a456-426614174001",
			TaskType:   "check",
			Repository: "s3:bucket/repo2",
			CreatedAt:  time.Now(),
		},
	}

	added, err := queue.EnqueueMultiple(tasks)
	if err == nil {
		t.Error("Expected error for duplicate task")
	}

	// Should have added 1 (the non-duplicate)
	if added != 1 {
		t.Errorf("Expected 1 task added, got %d", added)
	}

	// Total size should be 2
	if queue.Size() != 2 {
		t.Errorf("Expected queue size 2, got %d", queue.Size())
	}
}

// TestTaskQueueConcurrentAccess tests thread-safety (TDD - Epic 9.5)
func TestTaskQueueConcurrentAccess(t *testing.T) {
	queue := agent.NewTaskQueue()

	// Concurrent enqueues
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			task := agent.Task{
				TaskID:     "123e4567-e89b-12d3-a456-42661417400" + string(rune('0'+id)),
				PolicyID:   "234e5678-e89b-12d3-a456-426614174001",
				TaskType:   "backup",
				Repository: "s3:bucket/repo",
				CreatedAt:  time.Now(),
			}
			queue.Enqueue(task)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if queue.Size() != 10 {
		t.Errorf("Expected queue size 10, got %d", queue.Size())
	}
}
