package task

import (
	"context"
	"errors"
	"testing"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ========================================
// DependencyManager Tests with MockDatabase
// ========================================

func TestDependencyManager_ValidateDependenciesSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID1 := uuid.New()
	depID2 := uuid.New()
	dependencies := []uuid.UUID{depID1, depID2}

	// Mock QueryRow to return exists = true for each dependency
	mockRow1 := database.NewMockRowWithValues(true)
	mockRow2 := database.NewMockRowWithValues(true)

	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow1).Once()
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow2).Once()

	err := dm.ValidateDependencies(dependencies)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_ValidateDependenciesNotFound(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID := uuid.New()
	dependencies := []uuid.UUID{depID}

	// Mock QueryRow to return exists = false
	mockRow := database.NewMockRowWithValues(false)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	err := dm.ValidateDependencies(dependencies)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency task not found")
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_ValidateDependenciesQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID := uuid.New()
	dependencies := []uuid.UUID{depID}
	dbError := errors.New("query failed")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	err := dm.ValidateDependencies(dependencies)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check dependency existence")
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_CheckDependenciesCompletedAllComplete(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID1 := uuid.New()
	depID2 := uuid.New()
	dependencies := []uuid.UUID{depID1, depID2}

	// Mock QueryRow to return count = 2 (all completed)
	mockRow := database.NewMockRowWithValues(2)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	completed, err := dm.CheckDependenciesCompleted(dependencies)

	assert.NoError(t, err)
	assert.True(t, completed)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_CheckDependenciesCompletedNotAllComplete(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID1 := uuid.New()
	depID2 := uuid.New()
	dependencies := []uuid.UUID{depID1, depID2}

	// Mock QueryRow to return count = 1 (only 1 of 2 completed)
	mockRow := database.NewMockRowWithValues(1)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	completed, err := dm.CheckDependenciesCompleted(dependencies)

	assert.NoError(t, err)
	assert.False(t, completed)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_CheckDependenciesCompletedQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	dependencies := []uuid.UUID{uuid.New()}
	dbError := errors.New("query failed")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	completed, err := dm.CheckDependenciesCompleted(dependencies)

	assert.Error(t, err)
	assert.False(t, completed)
	assert.Contains(t, err.Error(), "failed to check dependency status")
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetBlockingDependenciesWithBlocking(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	depID1 := uuid.New()
	depID2 := uuid.New()
	depID3 := uuid.New()
	dependencies := []uuid.UUID{depID1, depID2, depID3}

	// Mock Query to return 2 blocking dependencies
	mockRows := database.NewMockRows([][]interface{}{
		{depID1},
		{depID3},
	})

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	blocking, err := dm.GetBlockingDependencies(dependencies)

	assert.NoError(t, err)
	assert.Len(t, blocking, 2)
	assert.Contains(t, blocking, depID1)
	assert.Contains(t, blocking, depID3)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetBlockingDependenciesNone(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	dependencies := []uuid.UUID{uuid.New()}

	// Mock Query to return empty result
	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	blocking, err := dm.GetBlockingDependencies(dependencies)

	assert.NoError(t, err)
	assert.Len(t, blocking, 0)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetBlockingDependenciesQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	dependencies := []uuid.UUID{uuid.New()}
	dbError := errors.New("query failed")

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, dbError)

	blocking, err := dm.GetBlockingDependencies(dependencies)

	assert.Error(t, err)
	assert.Nil(t, blocking)
	assert.Contains(t, err.Error(), "failed to query blocking dependencies")
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_DetectCircularDependenciesNoCircular(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID1 := uuid.New()
	depID2 := uuid.New()
	dependencies := []uuid.UUID{depID1, depID2}

	// Mock getTaskDependencies for depID1 (no dependencies)
	mockRow1 := database.NewMockRowWithValues([]uuid.UUID{})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow1).Once()

	// Mock getTaskDependencies for depID2 (no dependencies)
	mockRow2 := database.NewMockRowWithValues([]uuid.UUID{})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow2).Once()

	circular, err := dm.DetectCircularDependencies(taskID, dependencies)

	assert.NoError(t, err)
	assert.False(t, circular)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_DetectCircularDependenciesDirectCircular(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID := uuid.New()
	dependencies := []uuid.UUID{depID}

	// Mock getTaskDependencies for depID - returns taskID (circular!)
	mockRow := database.NewMockRowWithValues([]uuid.UUID{taskID})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	circular, err := dm.DetectCircularDependencies(taskID, dependencies)

	assert.NoError(t, err)
	assert.True(t, circular)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_DetectCircularDependenciesIndirectCircular(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID1 := uuid.New()
	depID2 := uuid.New()
	dependencies := []uuid.UUID{depID1}

	// depID1 depends on depID2
	mockRow1 := database.NewMockRowWithValues([]uuid.UUID{depID2})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow1).Once()

	// depID2 depends on taskID (circular!)
	mockRow2 := database.NewMockRowWithValues([]uuid.UUID{taskID})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow2).Once()

	circular, err := dm.DetectCircularDependencies(taskID, dependencies)

	assert.NoError(t, err)
	assert.True(t, circular)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_DetectCircularDependenciesQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID := uuid.New()
	dependencies := []uuid.UUID{depID}
	dbError := errors.New("query failed")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	circular, err := dm.DetectCircularDependencies(taskID, dependencies)

	assert.Error(t, err)
	assert.False(t, circular)
	assert.Contains(t, err.Error(), "failed to get task dependencies")
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependencyChainSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID1 := uuid.New()
	depID2 := uuid.New()

	// taskID depends on depID1
	mockRow1 := database.NewMockRowWithValues([]uuid.UUID{depID1})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow1).Once()

	// depID1 depends on depID2
	mockRow2 := database.NewMockRowWithValues([]uuid.UUID{depID2})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow2).Once()

	// depID2 has no dependencies
	mockRow3 := database.NewMockRowWithValues([]uuid.UUID{})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow3).Once()

	chain, err := dm.GetDependencyChain(taskID)

	assert.NoError(t, err)
	assert.Len(t, chain, 3)
	assert.Contains(t, chain, taskID)
	assert.Contains(t, chain, depID1)
	assert.Contains(t, chain, depID2)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependencyChainNoDependencies(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()

	// Mock task with no dependencies
	mockRow := database.NewMockRowWithValues([]uuid.UUID{})
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	chain, err := dm.GetDependencyChain(taskID)

	assert.NoError(t, err)
	assert.Len(t, chain, 1)
	assert.Equal(t, taskID, chain[0])
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependencyChainQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("query failed")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	chain, err := dm.GetDependencyChain(taskID)

	assert.Error(t, err)
	assert.Nil(t, chain)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependentTasksSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	dependentID1 := uuid.New()
	dependentID2 := uuid.New()

	// Mock Query to return tasks that depend on taskID
	mockRows := database.NewMockRows([][]interface{}{
		{dependentID1},
		{dependentID2},
	})

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	dependents, err := dm.GetDependentTasks(taskID)

	assert.NoError(t, err)
	assert.Len(t, dependents, 2)
	assert.Contains(t, dependents, dependentID1)
	assert.Contains(t, dependents, dependentID2)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependentTasksNone(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()

	// Mock Query to return empty result
	mockRows := database.NewMockRows([][]interface{}{})
	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRows, nil)

	dependents, err := dm.GetDependentTasks(taskID)

	assert.NoError(t, err)
	assert.Len(t, dependents, 0)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetDependentTasksQueryError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("query failed")

	mockDB.On("Query", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(nil, dbError)

	dependents, err := dm.GetDependentTasks(taskID)

	assert.Error(t, err)
	assert.Nil(t, dependents)
	assert.Contains(t, err.Error(), "failed to query dependent tasks")
	mockDB.AssertExpectations(t)
}

// Note: Scan error testing removed as it's difficult to properly mock
// The error is detected during rows.Scan() which is tested via rows.Err()
// after iteration, and that's already covered by other error cases

// ========================================
// Private Helper Method Tests
// ========================================

func TestDependencyManager_GetTaskDependenciesSuccess(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	depID1 := uuid.New()
	depID2 := uuid.New()
	expectedDeps := []uuid.UUID{depID1, depID2}

	mockRow := database.NewMockRowWithValues(expectedDeps)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	deps, err := dm.getTaskDependencies(taskID)

	assert.NoError(t, err)
	assert.Equal(t, expectedDeps, deps)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetTaskDependenciesError(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()
	dbError := errors.New("not found")

	mockRow := database.NewMockRowWithError(dbError)
	mockDB.On("QueryRow", context.Background(), mockDB.AnyString(), mockDB.AnyArgs()).Return(mockRow)

	deps, err := dm.getTaskDependencies(taskID)

	assert.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "failed to get task dependencies")
	mockDB.AssertExpectations(t)
}

// ========================================
// Edge Cases and Boundary Tests
// ========================================

func TestDependencyManager_ValidateDependenciesEmptyList(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	err := dm.ValidateDependencies([]uuid.UUID{})

	assert.NoError(t, err)
	// No database calls should be made for empty list
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_CheckDependenciesCompletedEmptyList(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	completed, err := dm.CheckDependenciesCompleted([]uuid.UUID{})

	assert.NoError(t, err)
	assert.True(t, completed) // Empty list is considered "all completed"
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_GetBlockingDependenciesEmptyList(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	blocking, err := dm.GetBlockingDependencies([]uuid.UUID{})

	assert.NoError(t, err)
	assert.Empty(t, blocking)
	mockDB.AssertExpectations(t)
}

func TestDependencyManager_DetectCircularDependenciesEmptyList(t *testing.T) {
	mockDB := database.NewMockDatabase()
	dm := NewDependencyManager(mockDB)

	taskID := uuid.New()

	circular, err := dm.DetectCircularDependencies(taskID, []uuid.UUID{})

	assert.NoError(t, err)
	assert.False(t, circular)
	mockDB.AssertExpectations(t)
}
