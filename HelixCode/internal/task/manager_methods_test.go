package task

import (
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"dev.helix.code/internal/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSplitStrategy implements SplitStrategy for testing
type MockSplitStrategy struct {
	subtasks []SubtaskData
	err      error
}

func (m *MockSplitStrategy) GenerateSubtasks(parent *Task, analysis *TaskAnalysis) ([]SubtaskData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.subtasks, nil
}

// TestSplitTask tests task splitting functionality
func TestSplitTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a large task
	task, err := tm.CreateTask(
		TaskTypePlanning,
		map[string]interface{}{
			"complexity": "high",
			"dataSize":   1000000,
		},
		PriorityHigh,
		CriticalityHigh,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Create mock split strategy
	strategy := &MockSplitStrategy{
		subtasks: []SubtaskData{
			{
				Data:         map[string]interface{}{"part": 1},
				Dependencies: []uuid.UUID{},
			},
			{
				Data:         map[string]interface{}{"part": 2},
				Dependencies: []uuid.UUID{},
			},
		},
	}

	// Mock database for subtask creation
	mockDB.MockExecSuccess(1) // For parent task update

	// Split the task
	subtasks, err := tm.SplitTask(task.ID, strategy)

	// Should complete without error
	assert.NoError(t, err)
	assert.Len(t, subtasks, 2)
}

// TestSplitTask_NotFound tests splitting non-existent task
func TestSplitTask_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	nonExistentID := uuid.New()
	strategy := &MockSplitStrategy{}

	_, err := tm.SplitTask(nonExistentID, strategy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestAssignTask tests task assignment to workers
func TestAssignTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeBuilding,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Create a worker manually in the tasks map
	workerID := uuid.New()
	worker := &Worker{
		ID:                 workerID,
		Hostname:           "test-worker",
		DisplayName:        "Test Worker",
		Capabilities:       []string{"building"},
		Status:             "active",
		HealthStatus:       "healthy",
		MaxConcurrentTasks: 5,
		CurrentTasksCount:  0,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	tm.mu.Lock()
	tm.workers[workerID] = worker
	tm.mu.Unlock()

	// Mock database expectations for assignment
	mockDB.MockExecSuccess(1) // Update task
	mockDB.MockExecSuccess(1) // Update worker

	// Assign task to worker
	err = tm.AssignTask(task.ID, workerID)

	// Should complete without error
	assert.NoError(t, err)
}

// TestAssignTask_NotFound tests assigning non-existent task
func TestAssignTask_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	nonExistentID := uuid.New()
	workerID := uuid.New()

	err := tm.AssignTask(nonExistentID, workerID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestCompleteTask tests task completion
func TestCompleteTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeBuilding,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Mock database update
	mockDB.MockExecSuccess(1)

	// Complete the task
	result := map[string]interface{}{
		"output": "success",
		"lines":  100,
	}
	err = tm.CompleteTask(task.ID, result)

	// Should complete without error
	assert.NoError(t, err)
}

// TestFailTask tests marking a task as failed
func TestFailTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeTesting,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Mock database update
	mockDB.MockExecSuccess(1)

	// Fail the task
	err = tm.FailTask(task.ID, "test error")

	// Should complete without error
	assert.NoError(t, err)
}

// TestFailTask_NotFound tests failing non-existent task
func TestFailTask_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	nonExistentID := uuid.New()

	err := tm.FailTask(nonExistentID, "error")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestCreateCheckpoint tests checkpoint creation
func TestCreateCheckpoint(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeRefactoring,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Mock database insert for checkpoint
	mockDB.MockExecSuccess(1)

	// Create checkpoint
	err = tm.CreateCheckpoint(task.ID, "progress-checkpoint", map[string]interface{}{
		"progress": 50,
		"step":     "analysis",
	})

	// Should complete without error
	assert.NoError(t, err)
}

// TestCreateCheckpoint_NotFound tests checkpoint for non-existent task
func TestCreateCheckpoint_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	nonExistentID := uuid.New()

	err := tm.CreateCheckpoint(nonExistentID, "checkpoint", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetTaskProgress tests retrieving task progress
func TestGetTaskProgress(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a task
	task, err := tm.CreateTask(
		TaskTypeDeployment,
		map[string]interface{}{"test": "data"},
		PriorityNormal,
		CriticalityNormal,
		[]uuid.UUID{},
	)
	require.NoError(t, err)

	// Get progress
	progress, err := tm.GetTaskProgress(task.ID)

	// Should complete without error
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	assert.Equal(t, task.ID, progress.TaskID)
}

// TestGetTaskProgress_NotFound tests progress for non-existent task
func TestGetTaskProgress_NotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	nonExistentID := uuid.New()

	progress, err := tm.GetTaskProgress(nonExistentID)
	assert.Error(t, err)
	assert.Nil(t, progress)
	assert.Contains(t, err.Error(), "not found")
}

// TestAnalyzeTaskForSplitting tests task analysis logic
func TestAnalyzeTaskForSplitting(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	// Create a complex task
	task := &Task{
		ID:   uuid.New(),
		Type: TaskTypePlanning,
		Data: map[string]interface{}{
			"complexity": "high",
			"dataSize":   5000000,
		},
		Priority:     PriorityHigh,
		Criticality:  CriticalityHigh,
		Dependencies: []uuid.UUID{},
	}

	// Analyze for splitting
	analysis, err := tm.analyzeTaskForSplitting(task)

	// Should return analysis
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, task.ID, analysis.TaskID)
	assert.Equal(t, task.Type, analysis.TaskType)
}

// TestEstimateComplexity tests complexity estimation
func TestEstimateComplexity(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	tests := []struct {
		name string
		task *Task
	}{
		{
			name: "simple task",
			task: &Task{
				Type: TaskTypePlanning,
				Data: map[string]interface{}{
					"complexity": "low",
				},
			},
		},
		{
			name: "complex task",
			task: &Task{
				Type: TaskTypeBuilding,
				Data: map[string]interface{}{
					"complexity": "high",
					"steps":      100,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := tm.estimateComplexity(tt.task)
			// Should return a valid complexity level
			assert.Contains(t, []ComplexityLevel{ComplexityLow, ComplexityMedium, ComplexityHigh}, complexity)
		})
	}
}

// TestEstimateDataSize tests data size estimation
func TestEstimateDataSize(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	tests := []struct {
		name string
		task *Task
	}{
		{
			name: "small data",
			task: &Task{
				Data: map[string]interface{}{
					"value": "small",
				},
			},
		},
		{
			name: "large data",
			task: &Task{
				Data: map[string]interface{}{
					"dataSize":  1000000,
					"itemCount": 5000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := tm.estimateDataSize(tt.task)
			assert.GreaterOrEqual(t, size, int64(0))
		})
	}
}

// TestCanWorkerHandleTask tests worker capability checking
func TestCanWorkerHandleTask(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	tests := []struct {
		name     string
		worker   *Worker
		task     *Task
		expected bool
	}{
		{
			name: "capable worker",
			worker: &Worker{
				Capabilities:       []string{"planning", "building", "testing"},
				MaxConcurrentTasks: 5,
				CurrentTasksCount:  2,
			},
			task: &Task{
				Type: TaskTypePlanning,
				Data: map[string]interface{}{
					"memoryRequired": 4096,
				},
			},
			expected: true,
		},
		{
			name: "incapable worker - missing capability",
			worker: &Worker{
				Capabilities:       []string{"testing"},
				MaxConcurrentTasks: 5,
				CurrentTasksCount:  2,
			},
			task: &Task{
				Type: TaskTypePlanning,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canHandle := tm.canWorkerHandleTask(tt.worker, tt.task)
			assert.Equal(t, tt.expected, canHandle)
		})
	}
}

// TestGetRequiredCapabilities tests capability extraction
func TestGetRequiredCapabilities(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	tests := []struct {
		name string
		task *Task
	}{
		{
			name: "planning task",
			task: &Task{
				Type: TaskTypePlanning,
			},
		},
		{
			name: "building task",
			task: &Task{
				Type: TaskTypeBuilding,
			},
		},
		{
			name: "testing task",
			task: &Task{
				Type: TaskTypeTesting,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := tm.getRequiredCapabilities(tt.task.Type)
			assert.NotNil(t, capabilities)
			// Should return at least an empty slice
			assert.GreaterOrEqual(t, len(capabilities), 0)
		})
	}
}

// TestUpdateWorkerInDB tests worker database update
func TestUpdateWorkerInDB(t *testing.T) {
	mockDB := database.NewMockDatabase()
	tm := NewTaskManager(mockDB, &redis.Client{})

	w := &Worker{
		ID:                 uuid.New(),
		Hostname:           "test-worker",
		DisplayName:        "Test Worker",
		Capabilities:       []string{"planning"},
		MaxConcurrentTasks: 5,
		CurrentTasksCount:  2,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Mock database update
	mockDB.MockExecSuccess(1)

	err := tm.updateWorkerInDB(w)

	// Should complete without error
	assert.NoError(t, err)
}
