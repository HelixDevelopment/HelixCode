package task

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// SplitTask intelligently splits a large task into subtasks
func (tm *TaskManager) SplitTask(parentTaskID uuid.UUID, strategy SplitStrategy) ([]*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	parentTask, exists := tm.tasks[parentTaskID]
	if !exists {
		return nil, fmt.Errorf("parent task not found: %s", parentTaskID)
	}

	// Analyze task for splitting
	analysis, err := tm.analyzeTaskForSplitting(parentTask)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze task: %v", err)
	}

	// Generate subtasks based on strategy
	subtasks, err := strategy.GenerateSubtasks(parentTask, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to generate subtasks: %v", err)
	}

	// Create subtasks
	var createdSubtasks []*Task
	for _, subtaskData := range subtasks {
		subtask, err := tm.CreateTask(
			parentTask.Type,
			subtaskData.Data,
			parentTask.Priority,
			parentTask.Criticality,
			subtaskData.Dependencies,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create subtask: %v", err)
		}
		createdSubtasks = append(createdSubtasks, subtask)
	}

	// Update parent task status
	parentTask.Status = TaskStatusWaitingForDeps
	parentTask.Data["subtasks"] = createdSubtasks
	tm.updateTaskInDB(parentTask)

	log.Printf("✅ Task %s split into %d subtasks", parentTaskID, len(createdSubtasks))
	return createdSubtasks, nil
}

// AssignTask assigns a task to a worker
func (tm *TaskManager) AssignTask(taskID uuid.UUID, workerID uuid.UUID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	worker, exists := tm.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found: %s", workerID)
	}

	// Check if worker can handle this task
	if !tm.canWorkerHandleTask(worker, task) {
		return fmt.Errorf("worker %s cannot handle task %s", workerID, taskID)
	}

	// Check worker capacity
	if worker.CurrentTasksCount >= worker.MaxConcurrentTasks {
		return fmt.Errorf("worker %s is at capacity", workerID)
	}

	// Update task
	task.AssignedWorker = &workerID
	task.Status = TaskStatusAssigned
	task.UpdatedAt = time.Now()

	// Update worker
	worker.CurrentTasksCount++
	worker.UpdatedAt = time.Now()

	// Update in database
	tm.updateTaskInDB(task)
	tm.updateWorkerInDB(worker)

	log.Printf("✅ Task %s assigned to worker %s", taskID, workerID)
	return nil
}

// CompleteTask marks a task as completed
func (tm *TaskManager) CompleteTask(taskID uuid.UUID, result map[string]interface{}) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Update task
	task.Status = TaskStatusCompleted
	task.ResultData = result
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	// Update worker if assigned
	if task.AssignedWorker != nil {
		if worker, exists := tm.workers[*task.AssignedWorker]; exists {
			worker.CurrentTasksCount--
			worker.UpdatedAt = now
			tm.updateWorkerInDB(worker)
		}
	}

	// Update in database
	tm.updateTaskInDB(task)

	log.Printf("✅ Task %s completed", taskID)
	return nil
}

// FailTask marks a task as failed
func (tm *TaskManager) FailTask(taskID uuid.UUID, errorMessage string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Check if we should retry
	if task.RetryCount < task.MaxRetries {
		task.RetryCount++
		task.Status = TaskStatusPending
		task.ErrorMessage = errorMessage
		task.AssignedWorker = nil
		task.UpdatedAt = time.Now()

		// Add back to queue
		tm.queue.AddTask(task)
		log.Printf("🔄 Task %s failed, retrying (attempt %d/%d)", taskID, task.RetryCount, task.MaxRetries)
	} else {
		task.Status = TaskStatusFailed
		task.ErrorMessage = errorMessage
		task.UpdatedAt = time.Now()
		log.Printf("❌ Task %s failed permanently", taskID)
	}

	// Update worker if assigned
	if task.AssignedWorker != nil {
		if worker, exists := tm.workers[*task.AssignedWorker]; exists {
			worker.CurrentTasksCount--
			worker.UpdatedAt = time.Now()
			tm.updateWorkerInDB(worker)
		}
	}

	// Update in database
	tm.updateTaskInDB(task)

	return nil
}

// CreateCheckpoint creates a checkpoint for a task
func (tm *TaskManager) CreateCheckpoint(taskID uuid.UUID, checkpointName string, checkpointData map[string]interface{}) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return tm.checkpointMgr.CreateCheckpoint(taskID, checkpointName, checkpointData)
}

// GetTaskProgress returns progress information for a task
func (tm *TaskManager) GetTaskProgress(taskID uuid.UUID) (*TaskProgress, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	progress := &TaskProgress{
		TaskID:    taskID,
		Status:    task.Status,
		Progress:  0.0,
		StartedAt: task.StartedAt,
		UpdatedAt: task.UpdatedAt,
	}

	// Calculate progress based on task type and status
	switch task.Status {
	case TaskStatusCompleted:
		progress.Progress = 100.0
	case TaskStatusRunning:
		// Estimate progress based on elapsed time vs estimated duration
		if task.StartedAt != nil && task.EstimatedDuration > 0 {
			elapsed := time.Since(*task.StartedAt)
			progress.Progress = float64(elapsed) / float64(task.EstimatedDuration) * 100
			if progress.Progress > 95 {
				progress.Progress = 95 // Cap at 95% until actually completed
			}
		} else {
			progress.Progress = 50.0 // Default estimate
		}
	case TaskStatusPending, TaskStatusAssigned:
		progress.Progress = 0.0
	}

	return progress, nil
}

// Helper methods

func (tm *TaskManager) analyzeTaskForSplitting(task *Task) (*TaskAnalysis, error) {
	// Analyze task data to determine optimal splitting strategy
	analysis := &TaskAnalysis{
		TaskID:       task.ID,
		TaskType:     task.Type,
		Complexity:   tm.estimateComplexity(task),
		DataSize:     tm.estimateDataSize(task),
		Dependencies: len(task.Dependencies),
	}

	return analysis, nil
}

func (tm *TaskManager) estimateComplexity(task *Task) ComplexityLevel {
	// Estimate task complexity based on type and data
	switch task.Type {
	case TaskTypePlanning, TaskTypeDesign:
		return ComplexityHigh
	case TaskTypeBuilding, TaskTypeRefactoring:
		return ComplexityMedium
	default:
		return ComplexityLow
	}
}

func (tm *TaskManager) estimateDataSize(task *Task) int64 {
	// Estimate data size based on task data
	dataSize := int64(0)
	if task.Data != nil {
		// Convert to JSON to estimate size
		jsonData, err := json.Marshal(task.Data)
		if err == nil {
			dataSize = int64(len(jsonData))
		}
	}
	return dataSize
}

func (tm *TaskManager) canWorkerHandleTask(worker *Worker, task *Task) bool {
	// Check if worker has required capabilities for task type
	requiredCaps := tm.getRequiredCapabilities(task.Type)
	for _, requiredCap := range requiredCaps {
		if !contains(worker.Capabilities, requiredCap) {
			return false
		}
	}
	return true
}

func (tm *TaskManager) getRequiredCapabilities(taskType TaskType) []string {
	// Define required capabilities for each task type
	switch taskType {
	case TaskTypeBuilding:
		return []string{"compilation", "build_tools"}
	case TaskTypeTesting:
		return []string{"test_execution", "coverage_analysis"}
	case TaskTypeRefactoring:
		return []string{"code_analysis", "refactoring_tools"}
	case TaskTypeDebugging:
		return []string{"debugging", "error_analysis"}
	default:
		return []string{"general_computation"}
	}
}

// Database operations

func (tm *TaskManager) storeTaskInDB(task *Task) error {
	// For now, just log the operation
	// In a real implementation, this would store the task in the database
	log.Printf("Storing task %s in database", task.ID)
	return nil
}

func (tm *TaskManager) updateTaskInDB(task *Task) error {
	// For now, just log the operation
	// In a real implementation, this would update the task in the database
	log.Printf("Updating task %s in database", task.ID)
	return nil
}

func (tm *TaskManager) updateWorkerInDB(worker *Worker) error {
	// For now, just log the operation
	// In a real implementation, this would update the worker in the database
	log.Printf("Updating worker %s in database", worker.ID)
	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
