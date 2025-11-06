package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================
// TASK CREATION AND INITIALIZATION TESTS
// ===========================================

func TestNewTask(t *testing.T) {
	title := "Test Task"
	description := "Test Description"
	priority := PriorityNormal

	task := NewTask(TaskTypePlanning, title, description, priority)

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, TaskTypePlanning, task.Type)
	assert.Equal(t, title, task.Title)
	assert.Equal(t, description, task.Description)
	assert.Equal(t, priority, task.Priority)
	assert.Equal(t, StatusPending, task.Status)
	assert.NotNil(t, task.Input)
	assert.NotNil(t, task.Output)
	assert.NotNil(t, task.Tags)
	assert.NotNil(t, task.Metadata)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestNewTaskAllTypes(t *testing.T) {
	types := []TaskType{
		TaskTypePlanning,
		TaskTypeAnalysis,
		TaskTypeCodeGeneration,
		TaskTypeCodeEdit,
		TaskTypeRefactoring,
		TaskTypeTesting,
		TaskTypeDebugging,
		TaskTypeReview,
		TaskTypeDocumentation,
		TaskTypeResearch,
	}

	for _, taskType := range types {
		task := NewTask(taskType, "Test", "Test", PriorityNormal)
		assert.Equal(t, taskType, task.Type)
	}
}

func TestNewTaskAllPriorities(t *testing.T) {
	priorities := []Priority{
		PriorityLow,
		PriorityNormal,
		PriorityHigh,
		PriorityCritical,
	}

	for _, priority := range priorities {
		task := NewTask(TaskTypePlanning, "Test", "Test", priority)
		assert.Equal(t, priority, task.Priority)
	}
}

// ===========================================
// TASK STATUS TRANSITION TESTS
// ===========================================

func TestTaskStart(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	agentID := "agent-123"

	// Initial state
	assert.Equal(t, StatusPending, task.Status)
	assert.Nil(t, task.StartedAt)
	assert.Empty(t, task.AssignedTo)

	// Start the task
	task.Start(agentID)

	assert.Equal(t, StatusInProgress, task.Status)
	assert.NotNil(t, task.StartedAt)
	assert.Equal(t, agentID, task.AssignedTo)
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskComplete(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.Start("agent-123")

	output := map[string]interface{}{
		"result": "success",
		"data":   []string{"item1", "item2"},
	}

	task.Complete(output)

	assert.Equal(t, StatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
	assert.Equal(t, output, task.Output)
	assert.Greater(t, task.Duration, time.Duration(0))
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskCompleteWithoutStart(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	output := map[string]interface{}{"result": "success"}
	task.Complete(output)

	assert.Equal(t, StatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
	assert.Equal(t, output, task.Output)
	// Duration should be 0 if never started
	assert.Equal(t, time.Duration(0), task.Duration)
}

func TestTaskFail(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.Start("agent-123")

	reason := "Failed due to timeout"
	task.Fail(reason)

	assert.Equal(t, StatusFailed, task.Status)
	assert.NotNil(t, task.Metadata)
	assert.Equal(t, reason, task.Metadata["failure_reason"])
	assert.NotNil(t, task.Metadata["failed_at"])
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskFailWithNilMetadata(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.Metadata = nil

	reason := "Test failure"
	task.Fail(reason)

	assert.Equal(t, StatusFailed, task.Status)
	assert.NotNil(t, task.Metadata)
	assert.Equal(t, reason, task.Metadata["failure_reason"])
}

func TestTaskBlock(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	reason := "Waiting for dependency"
	blockedBy := []string{"task-456", "task-789"}

	task.Block(reason, blockedBy)

	assert.Equal(t, StatusBlocked, task.Status)
	assert.Equal(t, blockedBy, task.BlockedBy)
	assert.NotNil(t, task.Metadata)
	assert.Equal(t, reason, task.Metadata["block_reason"])
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskUnblock(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	// First block the task
	task.Block("Test block", []string{"task-456"})
	assert.Equal(t, StatusBlocked, task.Status)
	assert.NotEmpty(t, task.BlockedBy)

	// Now unblock it
	task.Unblock()

	assert.Equal(t, StatusReady, task.Status)
	assert.Empty(t, task.BlockedBy)
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskUnblockWhenNotBlocked(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	initialStatus := task.Status
	task.Unblock()

	// Status should remain unchanged
	assert.Equal(t, initialStatus, task.Status)
}

func TestTaskCancel(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.Start("agent-123")

	reason := "User cancelled"
	task.Cancel(reason)

	assert.Equal(t, StatusCancelled, task.Status)
	assert.NotNil(t, task.Metadata)
	assert.Equal(t, reason, task.Metadata["cancellation_reason"])
	assert.False(t, task.UpdatedAt.IsZero())
}

// ===========================================
// TASK STATE CHECK TESTS
// ===========================================

func TestTaskIsReady(t *testing.T) {
	tests := []struct {
		name           string
		status         TaskStatus
		blocked        []string
		depends        []string
		completed      map[string]bool
		expectedReady  bool
	}{
		{
			name:           "Ready with no dependencies",
			status:         StatusPending,
			blocked:        []string{},
			depends:        []string{},
			completed:      map[string]bool{},
			expectedReady:  true,
		},
		{
			name:           "Ready with completed dependencies",
			status:         StatusPending,
			blocked:        []string{},
			depends:        []string{"task-1", "task-2"},
			completed:      map[string]bool{"task-1": true, "task-2": true},
			expectedReady:  true,
		},
		{
			name:           "Not ready - incomplete dependencies",
			status:         StatusPending,
			blocked:        []string{},
			depends:        []string{"task-1", "task-2"},
			completed:      map[string]bool{"task-1": true},
			expectedReady:  false,
		},
		{
			name:           "Not ready - blocked",
			status:         StatusPending,
			blocked:        []string{"blocker-1"},
			depends:        []string{},
			completed:      map[string]bool{},
			expectedReady:  false,
		},
		{
			name:           "Not ready - already completed",
			status:         StatusCompleted,
			blocked:        []string{},
			depends:        []string{},
			completed:      map[string]bool{},
			expectedReady:  false,
		},
		{
			name:           "Not ready - in progress",
			status:         StatusInProgress,
			blocked:        []string{},
			depends:        []string{},
			completed:      map[string]bool{},
			expectedReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
			task.Status = tt.status
			task.BlockedBy = tt.blocked
			task.DependsOn = tt.depends

			result := task.IsReady(tt.completed)
			assert.Equal(t, tt.expectedReady, result)
		})
	}
}

func TestTaskCanStart(t *testing.T) {
	tests := []struct {
		name        string
		status      TaskStatus
		expectedCan bool
	}{
		{"Can start - pending", StatusPending, true},
		{"Can start - ready", StatusReady, true},
		{"Cannot start - in progress", StatusInProgress, false},
		{"Cannot start - completed", StatusCompleted, false},
		{"Cannot start - failed", StatusFailed, false},
		{"Cannot start - cancelled", StatusCancelled, false},
		{"Cannot start - blocked", StatusBlocked, false},
		{"Cannot start - assigned", StatusAssigned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
			task.Status = tt.status

			result := task.CanStart()
			assert.Equal(t, tt.expectedCan, result)
		})
	}
}

func TestTaskIsCompleted(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	assert.False(t, task.IsCompleted())

	task.Complete(map[string]interface{}{"result": "success"})
	assert.True(t, task.IsCompleted())
}

func TestTaskIsFailed(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	assert.False(t, task.IsFailed())

	task.Fail("Test failure")
	assert.True(t, task.IsFailed())
}

func TestTaskIsActive(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	assert.False(t, task.IsActive())

	task.Start("agent-123")
	assert.True(t, task.IsActive())

	task.Complete(map[string]interface{}{})
	assert.False(t, task.IsActive())
}

// ===========================================
// RESULT TESTS
// ===========================================

func TestNewResult(t *testing.T) {
	taskID := "task-123"
	agentID := "agent-456"

	result := NewResult(taskID, agentID)

	assert.Equal(t, taskID, result.TaskID)
	assert.Equal(t, agentID, result.AgentID)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Output)
	assert.NotNil(t, result.Artifacts)
	assert.Equal(t, 0.0, result.Confidence)
	assert.False(t, result.Timestamp.IsZero())
}

func TestResultSetSuccess(t *testing.T) {
	result := NewResult("task-123", "agent-456")

	output := map[string]interface{}{
		"code": "package main",
		"lines": 100,
	}
	confidence := 0.95

	result.SetSuccess(output, confidence)

	assert.True(t, result.Success)
	assert.Equal(t, output, result.Output)
	assert.Equal(t, confidence, result.Confidence)
	assert.Empty(t, result.Error)
}

func TestResultSetFailure(t *testing.T) {
	result := NewResult("task-123", "agent-456")

	testErr := assert.AnError

	result.SetFailure(testErr)

	assert.False(t, result.Success)
	assert.Equal(t, testErr.Error(), result.Error)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestResultSetFailureWithNil(t *testing.T) {
	result := NewResult("task-123", "agent-456")

	result.SetFailure(nil)

	assert.False(t, result.Success)
	assert.Empty(t, result.Error)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestResultAddArtifact(t *testing.T) {
	result := NewResult("task-123", "agent-456")

	artifact1 := Artifact{
		ID:      "artifact-1",
		Type:    "code",
		Path:    "/path/to/file.go",
		Content: "package main",
		Size:    100,
	}
	artifact2 := Artifact{
		ID:      "artifact-2",
		Type:    "test",
		Path:    "/path/to/file_test.go",
		Content: "package main",
		Size:    200,
	}

	assert.Empty(t, result.Artifacts)

	result.AddArtifact(artifact1)
	assert.Len(t, result.Artifacts, 1)
	assert.Equal(t, artifact1, result.Artifacts[0])

	result.AddArtifact(artifact2)
	assert.Len(t, result.Artifacts, 2)
	assert.Equal(t, artifact2, result.Artifacts[1])
}

// ===========================================
// TASK DEPENDENCY TESTS
// ===========================================

func TestTaskWithDependencies(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.DependsOn = []string{"task-1", "task-2", "task-3"}

	// None complete
	assert.False(t, task.IsReady(map[string]bool{}))

	// Partial complete
	assert.False(t, task.IsReady(map[string]bool{
		"task-1": true,
		"task-2": true,
	}))

	// All complete
	assert.True(t, task.IsReady(map[string]bool{
		"task-1": true,
		"task-2": true,
		"task-3": true,
	}))
}

func TestTaskLifecycle(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test Task", "Test Description", PriorityHigh)

	// 1. Initial state
	assert.Equal(t, StatusPending, task.Status)
	assert.False(t, task.IsActive())
	assert.False(t, task.IsCompleted())
	assert.False(t, task.IsFailed())

	// 2. Start
	assert.True(t, task.CanStart())
	task.Start("agent-123")
	assert.Equal(t, StatusInProgress, task.Status)
	assert.True(t, task.IsActive())

	// 3. Complete
	output := map[string]interface{}{"result": "success"}
	task.Complete(output)
	assert.Equal(t, StatusCompleted, task.Status)
	assert.False(t, task.IsActive())
	assert.True(t, task.IsCompleted())
	assert.Equal(t, output, task.Output)
}

func TestTaskFailureLifecycle(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test Task", "Test Description", PriorityHigh)

	// Start
	task.Start("agent-123")
	assert.True(t, task.IsActive())

	// Fail
	task.Fail("Timeout occurred")
	assert.False(t, task.IsActive())
	assert.True(t, task.IsFailed())
	assert.Equal(t, StatusFailed, task.Status)
	assert.Equal(t, "Timeout occurred", task.Metadata["failure_reason"])
}

func TestTaskBlockUnblockLifecycle(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test Task", "Test Description", PriorityHigh)

	// Block
	task.Block("Waiting for resource", []string{"resource-1"})
	assert.Equal(t, StatusBlocked, task.Status)
	assert.Contains(t, task.BlockedBy, "resource-1")
	assert.False(t, task.CanStart())

	// Unblock
	task.Unblock()
	assert.Equal(t, StatusReady, task.Status)
	assert.Empty(t, task.BlockedBy)
	assert.True(t, task.CanStart())
}

// ===========================================
// TASK METRICS TESTS
// ===========================================

func TestTaskMetrics(t *testing.T) {
	metrics := &TaskMetrics{
		TokensUsed:      1000,
		LLMCalls:        5,
		ToolCalls:       3,
		FilesModified:   2,
		LinesAdded:      150,
		LinesRemoved:    50,
		TestsGenerated:  10,
		ExecutionTime:   5 * time.Second,
	}

	result := NewResult("task-123", "agent-456")
	result.Metrics = metrics

	assert.Equal(t, 1000, result.Metrics.TokensUsed)
	assert.Equal(t, 5, result.Metrics.LLMCalls)
	assert.Equal(t, 3, result.Metrics.ToolCalls)
	assert.Equal(t, 2, result.Metrics.FilesModified)
	assert.Equal(t, 150, result.Metrics.LinesAdded)
	assert.Equal(t, 50, result.Metrics.LinesRemoved)
	assert.Equal(t, 10, result.Metrics.TestsGenerated)
	assert.Equal(t, 5*time.Second, result.Metrics.ExecutionTime)
}

// ===========================================
// ARTIFACT TESTS
// ===========================================

func TestArtifactTypes(t *testing.T) {
	types := []string{"code", "test", "doc", "config"}

	for _, artifactType := range types {
		artifact := Artifact{
			ID:        "artifact-1",
			Type:      artifactType,
			Path:      "/path/to/file",
			Content:   "content",
			Size:      100,
			Checksum:  "abc123",
			CreatedAt: time.Now(),
		}

		assert.Equal(t, artifactType, artifact.Type)
		assert.NotEmpty(t, artifact.ID)
		assert.NotEmpty(t, artifact.Path)
		assert.NotEmpty(t, artifact.Content)
		assert.Greater(t, artifact.Size, int64(0))
		assert.False(t, artifact.CreatedAt.IsZero())
	}
}

// ===========================================
// EDGE CASE TESTS
// ===========================================

func TestTaskMultipleStateTransitions(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	// Pending -> Blocked -> Ready -> In Progress -> Completed
	task.Block("Waiting", []string{"dep-1"})
	assert.Equal(t, StatusBlocked, task.Status)

	task.Unblock()
	assert.Equal(t, StatusReady, task.Status)

	task.Start("agent-123")
	assert.Equal(t, StatusInProgress, task.Status)

	task.Complete(map[string]interface{}{"done": true})
	assert.Equal(t, StatusCompleted, task.Status)
}

func TestTaskCancelFromDifferentStates(t *testing.T) {
	states := []TaskStatus{
		StatusPending,
		StatusReady,
		StatusAssigned,
		StatusInProgress,
		StatusBlocked,
	}

	for _, state := range states {
		task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
		task.Status = state

		task.Cancel("User requested")
		assert.Equal(t, StatusCancelled, task.Status)
		assert.Equal(t, "User requested", task.Metadata["cancellation_reason"])
	}
}

func TestTaskWithEmptyDependencies(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)
	task.DependsOn = []string{}

	assert.True(t, task.IsReady(map[string]bool{}))
	assert.True(t, task.IsReady(map[string]bool{"other-task": true}))
}

func TestResultWithComplexOutput(t *testing.T) {
	result := NewResult("task-123", "agent-456")

	complexOutput := map[string]interface{}{
		"string":  "value",
		"number":  42,
		"boolean": true,
		"array":   []int{1, 2, 3},
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	result.SetSuccess(complexOutput, 0.98)

	assert.True(t, result.Success)
	assert.Equal(t, "value", result.Output["string"])
	assert.Equal(t, 42, result.Output["number"])
	assert.Equal(t, true, result.Output["boolean"])
	assert.Equal(t, []int{1, 2, 3}, result.Output["array"])

	nested, ok := result.Output["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value1", nested["key1"])
	assert.Equal(t, 123, nested["key2"])
}

func TestTaskDurationCalculation(t *testing.T) {
	task := NewTask(TaskTypePlanning, "Test", "Test", PriorityNormal)

	task.Start("agent-123")
	startTime := *task.StartedAt

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	task.Complete(map[string]interface{}{})

	assert.Greater(t, task.Duration, time.Duration(0))
	assert.True(t, task.CompletedAt.After(startTime))
}
