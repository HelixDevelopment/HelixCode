package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================================
// CheckpointManager Tests with MockDatabase
// ========================================

func TestCheckpointManager_CreateCheckpointSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	checkpointName := "step-1-completed"
	checkpointData := map[string]interface{}{
		"progress": 50,
		"status":   "completed",
	}

	// Mock successful Exec
	mockDB.MockExecSuccess(1)

	err := cm.CreateCheckpoint(taskID, checkpointName, checkpointData)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_CreateCheckpointMarshalError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	checkpointName := "test-checkpoint"

	// Create data that cannot be marshaled (channels, functions, etc.)
	// Using a channel which cannot be JSON marshaled
	invalidData := map[string]interface{}{
		"channel": make(chan int),
	}

	err := cm.CreateCheckpoint(taskID, checkpointName, invalidData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal checkpoint data")
}

func TestCheckpointManager_CreateCheckpointDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	checkpointName := "test-checkpoint"
	checkpointData := map[string]interface{}{"key": "value"}

	dbError := errors.New("database connection lost")
	mockDB.MockExecError(dbError)

	err := cm.CreateCheckpoint(taskID, checkpointName, checkpointData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create checkpoint")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetCheckpointsSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	now := time.Now()

	// Create checkpoint data
	checkpointData := map[string]interface{}{"progress": 75}
	checkpointDataJSON, _ := json.Marshal(checkpointData)

	checkpointID1 := uuid.New()
	checkpointID2 := uuid.New()
	workerID := uuid.New()

	// Mock rows with 2 checkpoints
	mockRows := database.NewMockRows([][]interface{}{
		{checkpointID1, "checkpoint-1", checkpointDataJSON, workerID, now},
		{checkpointID2, "checkpoint-2", checkpointDataJSON, workerID, now.Add(-time.Hour)},
	})

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	checkpoints, err := cm.GetCheckpoints(taskID)

	assert.NoError(t, err)
	assert.Len(t, checkpoints, 2)
	assert.Equal(t, checkpointID1, checkpoints[0].ID)
	assert.Equal(t, "checkpoint-1", checkpoints[0].CheckpointName)
	assert.Equal(t, float64(75), checkpoints[0].CheckpointData["progress"])
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetCheckpointsEmpty(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()

	// Mock empty result
	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	checkpoints, err := cm.GetCheckpoints(taskID)

	assert.NoError(t, err)
	assert.Len(t, checkpoints, 0)
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetCheckpointsQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("query failed")

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, dbError)

	checkpoints, err := cm.GetCheckpoints(taskID)

	assert.Error(t, err)
	assert.Nil(t, checkpoints)
	assert.Contains(t, err.Error(), "failed to query checkpoints")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetCheckpointsUnmarshalError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	now := time.Now()

	checkpointID := uuid.New()
	workerID := uuid.New()
	invalidJSON := []byte("{invalid json}")

	// Mock rows with invalid JSON
	mockRows := database.NewMockRows([][]interface{}{
		{checkpointID, "checkpoint-1", invalidJSON, workerID, now},
	})

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	checkpoints, err := cm.GetCheckpoints(taskID)

	assert.Error(t, err)
	assert.Nil(t, checkpoints)
	assert.Contains(t, err.Error(), "failed to unmarshal checkpoint data")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetLatestCheckpointSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	checkpointID := uuid.New()
	workerID := uuid.New()
	now := time.Now()

	checkpointData := map[string]interface{}{"step": "final"}
	checkpointDataJSON, _ := json.Marshal(checkpointData)

	// Mock successful QueryRow
	mockRow := database.NewMockRowWithValues(
		checkpointID,
		"latest-checkpoint",
		checkpointDataJSON,
		workerID,
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	checkpoint, err := cm.GetLatestCheckpoint(taskID)

	assert.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, checkpointID, checkpoint.ID)
	assert.Equal(t, "latest-checkpoint", checkpoint.CheckpointName)
	assert.Equal(t, "final", checkpoint.CheckpointData["step"])
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetLatestCheckpointNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("no rows")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	checkpoint, err := cm.GetLatestCheckpoint(taskID)

	assert.Error(t, err)
	assert.Nil(t, checkpoint)
	assert.Contains(t, err.Error(), "failed to get latest checkpoint")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_GetLatestCheckpointUnmarshalError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	checkpointID := uuid.New()
	workerID := uuid.New()
	now := time.Now()
	invalidJSON := []byte("{bad json")

	mockRow := database.NewMockRowWithValues(
		checkpointID,
		"checkpoint",
		invalidJSON,
		workerID,
		now,
	)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	checkpoint, err := cm.GetLatestCheckpoint(taskID)

	assert.Error(t, err)
	assert.Nil(t, checkpoint)
	assert.Contains(t, err.Error(), "failed to unmarshal checkpoint data")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteCheckpointSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	checkpointID := uuid.New()

	// Mock successful deletion
	mockDB.MockExecSuccess(1)

	err := cm.DeleteCheckpoint(checkpointID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteCheckpointNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	checkpointID := uuid.New()

	// Mock no rows affected - still returns success (no error check for rows affected)
	mockDB.MockExecSuccess(0)

	err := cm.DeleteCheckpoint(checkpointID)

	assert.NoError(t, err) // Function doesn't check RowsAffected()
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteCheckpointDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	checkpointID := uuid.New()
	dbError := errors.New("connection lost")

	mockDB.MockExecError(dbError)

	err := cm.DeleteCheckpoint(checkpointID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete checkpoint")
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteAllCheckpointsSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()

	// Mock successful deletion of multiple checkpoints
	mockDB.MockExecSuccess(5)

	err := cm.DeleteAllCheckpoints(taskID)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteAllCheckpointsNoneFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()

	// Mock no rows affected (no checkpoints exist) - still returns success
	mockDB.MockExecSuccess(0)

	err := cm.DeleteAllCheckpoints(taskID)

	assert.NoError(t, err) // Function doesn't check RowsAffected()
	mockDB.AssertExpectations(t)
}

func TestCheckpointManager_DeleteAllCheckpointsDatabaseError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	cm := NewCheckpointManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("database error")

	mockDB.MockExecError(dbError)

	err := cm.DeleteAllCheckpoints(taskID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete checkpoints")
	mockDB.AssertExpectations(t)
}

// ========================================
// Checkpoint Struct Tests
// ========================================

func TestCheckpoint_JSONMarshaling(t *testing.T) {
	checkpoint := Checkpoint{
		ID:             uuid.New(),
		CheckpointName: "test-checkpoint",
		CheckpointData: map[string]interface{}{
			"progress": 100,
			"message":  "complete",
		},
		WorkerID:  uuid.New(),
		CreatedAt: time.Now(),
	}

	// Test marshaling
	data, err := json.Marshal(checkpoint)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Test unmarshaling
	var unmarshaled Checkpoint
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, checkpoint.ID, unmarshaled.ID)
	assert.Equal(t, checkpoint.CheckpointName, unmarshaled.CheckpointName)
}
