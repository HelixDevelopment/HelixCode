package task

import (
	"sort"
)

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		highPriority:   make([]*Task, 0),
		normalPriority: make([]*Task, 0),
		lowPriority:    make([]*Task, 0),
	}
}

// AddTask adds a task to the appropriate queue based on priority
func (tq *TaskQueue) AddTask(task *Task) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	switch task.Priority {
	case PriorityCritical, PriorityHigh:
		tq.highPriority = append(tq.highPriority, task)
		// Sort high priority tasks by criticality and priority
		tq.sortHighPriorityTasks()
	case PriorityNormal:
		tq.normalPriority = append(tq.normalPriority, task)
	case PriorityLow:
		tq.lowPriority = append(tq.lowPriority, task)
	}
}

// GetNextTask returns the next task to be processed
func (tq *TaskQueue) GetNextTask() *Task {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Check high priority tasks first
	if len(tq.highPriority) > 0 {
		task := tq.highPriority[0]
		tq.highPriority = tq.highPriority[1:]
		return task
	}

	// Check normal priority tasks
	if len(tq.normalPriority) > 0 {
		task := tq.normalPriority[0]
		tq.normalPriority = tq.normalPriority[1:]
		return task
	}

	// Check low priority tasks
	if len(tq.lowPriority) > 0 {
		task := tq.lowPriority[0]
		tq.lowPriority = tq.lowPriority[1:]
		return task
	}

	return nil
}

// RemoveTask removes a specific task from the queue
func (tq *TaskQueue) RemoveTask(taskID string) bool {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Try to remove from high priority
	if removed := tq.removeFromSlice(&tq.highPriority, taskID); removed {
		return true
	}

	// Try to remove from normal priority
	if removed := tq.removeFromSlice(&tq.normalPriority, taskID); removed {
		return true
	}

	// Try to remove from low priority
	if removed := tq.removeFromSlice(&tq.lowPriority, taskID); removed {
		return true
	}

	return false
}

// GetQueueStats returns statistics about the queue
func (tq *TaskQueue) GetQueueStats() QueueStats {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return QueueStats{
		HighPriority:   len(tq.highPriority),
		NormalPriority: len(tq.normalPriority),
		LowPriority:    len(tq.lowPriority),
		Total:          len(tq.highPriority) + len(tq.normalPriority) + len(tq.lowPriority),
	}
}

// Clear clears all tasks from the queue
func (tq *TaskQueue) Clear() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.highPriority = make([]*Task, 0)
	tq.normalPriority = make([]*Task, 0)
	tq.lowPriority = make([]*Task, 0)
}

// Helper methods

func (tq *TaskQueue) sortHighPriorityTasks() {
	sort.Slice(tq.highPriority, func(i, j int) bool {
		taskA := tq.highPriority[i]
		taskB := tq.highPriority[j]

		// First sort by criticality
		if taskA.Criticality != taskB.Criticality {
			return tq.getCriticalityWeight(taskA.Criticality) > tq.getCriticalityWeight(taskB.Criticality)
		}

		// Then sort by priority
		return taskA.Priority > taskB.Priority
	})
}

func (tq *TaskQueue) getCriticalityWeight(criticality TaskCriticality) int {
	switch criticality {
	case CriticalityCritical:
		return 4
	case CriticalityHigh:
		return 3
	case CriticalityNormal:
		return 2
	case CriticalityLow:
		return 1
	default:
		return 0
	}
}

func (tq *TaskQueue) removeFromSlice(slice *[]*Task, taskID string) bool {
	for i, task := range *slice {
		if task.ID.String() == taskID {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return true
		}
	}
	return false
}

// QueueStats represents queue statistics
type QueueStats struct {
	HighPriority   int `json:"high_priority"`
	NormalPriority int `json:"normal_priority"`
	LowPriority    int `json:"low_priority"`
	Total          int `json:"total"`
}
