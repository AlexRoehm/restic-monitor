package agent

import (
	"fmt"
	"sync"
)

// TaskQueue manages pending tasks for the agent
type TaskQueue struct {
	mu      sync.RWMutex
	tasks   []Task
	taskIDs map[string]bool // For duplicate detection
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:   make([]Task, 0),
		taskIDs: make(map[string]bool),
	}
}

// Enqueue adds a task to the queue
// Returns error if task with same ID already exists
func (tq *TaskQueue) Enqueue(task Task) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check for duplicate
	if tq.taskIDs[task.TaskID] {
		return fmt.Errorf("task %s already in queue", task.TaskID)
	}

	// Add to queue
	tq.tasks = append(tq.tasks, task)
	tq.taskIDs[task.TaskID] = true

	return nil
}

// EnqueueMultiple adds multiple tasks to the queue
// Skips duplicates and returns count of tasks added
func (tq *TaskQueue) EnqueueMultiple(tasks []Task) (int, error) {
	added := 0
	var errors []string

	for _, task := range tasks {
		err := tq.Enqueue(task)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		added++
	}

	if len(errors) > 0 {
		return added, fmt.Errorf("enqueue errors: %v", errors)
	}

	return added, nil
}

// Dequeue removes and returns the next task from the queue
// Returns nil if queue is empty
func (tq *TaskQueue) Dequeue() *Task {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if len(tq.tasks) == 0 {
		return nil
	}

	// Get first task
	task := tq.tasks[0]

	// Remove from queue
	tq.tasks = tq.tasks[1:]
	delete(tq.taskIDs, task.TaskID)

	return &task
}

// Peek returns the next task without removing it
// Returns nil if queue is empty
func (tq *TaskQueue) Peek() *Task {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	if len(tq.tasks) == 0 {
		return nil
	}

	task := tq.tasks[0]
	return &task
}

// Size returns the number of tasks in the queue
func (tq *TaskQueue) Size() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return len(tq.tasks)
}

// IsEmpty returns true if the queue has no tasks
func (tq *TaskQueue) IsEmpty() bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return len(tq.tasks) == 0
}

// Contains checks if a task with the given ID is in the queue
func (tq *TaskQueue) Contains(taskID string) bool {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return tq.taskIDs[taskID]
}

// Clear removes all tasks from the queue
func (tq *TaskQueue) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.tasks = make([]Task, 0)
	tq.taskIDs = make(map[string]bool)
}

// GetAll returns a copy of all tasks in the queue
func (tq *TaskQueue) GetAll() []Task {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	// Return a copy to prevent external modification
	tasksCopy := make([]Task, len(tq.tasks))
	copy(tasksCopy, tq.tasks)

	return tasksCopy
}

// Remove removes a specific task by ID
// Returns true if task was found and removed
func (tq *TaskQueue) Remove(taskID string) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Find task index
	index := -1
	for i, task := range tq.tasks {
		if task.TaskID == taskID {
			index = i
			break
		}
	}

	if index == -1 {
		return false // Task not found
	}

	// Remove from queue
	tq.tasks = append(tq.tasks[:index], tq.tasks[index+1:]...)
	delete(tq.taskIDs, taskID)

	return true
}
