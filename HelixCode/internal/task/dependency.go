package task

import (
	"context"
	"fmt"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
)

// NewDependencyManager creates a new dependency manager
func NewDependencyManager(db *database.Database) *DependencyManager {
	return &DependencyManager{
		db: db,
	}
}

// ValidateDependencies validates that all dependencies exist and are valid
func (dm *DependencyManager) ValidateDependencies(dependencies []uuid.UUID) error {
	if len(dependencies) == 0 {
		return nil
	}

	ctx := context.Background()

	// Check if all dependencies exist
	for _, depID := range dependencies {
		var exists bool
		err := dm.db.Pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM distributed_tasks WHERE id = $1)
		`, depID).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check dependency existence: %v", err)
		}

		if !exists {
			return fmt.Errorf("dependency task not found: %s", depID)
		}
	}

	return nil
}

// CheckDependenciesCompleted checks if all dependencies are completed
func (dm *DependencyManager) CheckDependenciesCompleted(dependencies []uuid.UUID) (bool, error) {
	if len(dependencies) == 0 {
		return true, nil
	}

	ctx := context.Background()

	// Count completed dependencies
	var completedCount int
	err := dm.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM distributed_tasks 
		WHERE id = ANY($1) AND status = 'completed'
	`, dependencies).Scan(&completedCount)

	if err != nil {
		return false, fmt.Errorf("failed to check dependency status: %v", err)
	}

	return completedCount == len(dependencies), nil
}

// GetBlockingDependencies returns dependencies that are not completed
func (dm *DependencyManager) GetBlockingDependencies(dependencies []uuid.UUID) ([]uuid.UUID, error) {
	if len(dependencies) == 0 {
		return []uuid.UUID{}, nil
	}

	ctx := context.Background()

	rows, err := dm.db.Pool.Query(ctx, `
		SELECT id FROM distributed_tasks 
		WHERE id = ANY($1) AND status != 'completed'
	`, dependencies)

	if err != nil {
		return nil, fmt.Errorf("failed to query blocking dependencies: %v", err)
	}
	defer rows.Close()

	var blockingDeps []uuid.UUID
	for rows.Next() {
		var depID uuid.UUID
		if err := rows.Scan(&depID); err != nil {
			return nil, fmt.Errorf("failed to scan dependency ID: %v", err)
		}
		blockingDeps = append(blockingDeps, depID)
	}

	return blockingDeps, nil
}

// DetectCircularDependencies detects circular dependencies in the task graph
func (dm *DependencyManager) DetectCircularDependencies(taskID uuid.UUID, dependencies []uuid.UUID) (bool, error) {
	if len(dependencies) == 0 {
		return false, nil
	}

	// Build dependency graph
	graph := make(map[uuid.UUID][]uuid.UUID)

	// Add current task dependencies
	graph[taskID] = dependencies

	// Check each dependency for circular references
	for _, depID := range dependencies {
		if circular, err := dm.checkCircularDependency(depID, taskID, make(map[uuid.UUID]bool)); err != nil {
			return false, err
		} else if circular {
			return true, nil
		}
	}

	return false, nil
}

// GetDependencyChain returns the complete dependency chain for a task
func (dm *DependencyManager) GetDependencyChain(taskID uuid.UUID) ([]uuid.UUID, error) {
	visited := make(map[uuid.UUID]bool)
	chain := make([]uuid.UUID, 0)

	if err := dm.traverseDependencies(taskID, visited, &chain); err != nil {
		return nil, err
	}

	return chain, nil
}

// GetDependentTasks returns all tasks that depend on the given task
func (dm *DependencyManager) GetDependentTasks(taskID uuid.UUID) ([]uuid.UUID, error) {
	ctx := context.Background()

	rows, err := dm.db.Pool.Query(ctx, `
		SELECT id FROM distributed_tasks 
		WHERE $1 = ANY(dependencies)
	`, taskID)

	if err != nil {
		return nil, fmt.Errorf("failed to query dependent tasks: %v", err)
	}
	defer rows.Close()

	var dependentTasks []uuid.UUID
	for rows.Next() {
		var taskID uuid.UUID
		if err := rows.Scan(&taskID); err != nil {
			return nil, fmt.Errorf("failed to scan task ID: %v", err)
		}
		dependentTasks = append(dependentTasks, taskID)
	}

	return dependentTasks, nil
}

// Helper methods

func (dm *DependencyManager) checkCircularDependency(currentID uuid.UUID, targetID uuid.UUID, visited map[uuid.UUID]bool) (bool, error) {
	if visited[currentID] {
		return false, nil // Already visited this path
	}

	visited[currentID] = true

	// Get dependencies of current task
	dependencies, err := dm.getTaskDependencies(currentID)
	if err != nil {
		return false, err
	}

	// Check if target ID is in dependencies
	for _, depID := range dependencies {
		if depID == targetID {
			return true, nil // Circular dependency found
		}

		// Recursively check dependencies
		if circular, err := dm.checkCircularDependency(depID, targetID, visited); err != nil {
			return false, err
		} else if circular {
			return true, nil
		}
	}

	return false, nil
}

func (dm *DependencyManager) getTaskDependencies(taskID uuid.UUID) ([]uuid.UUID, error) {
	ctx := context.Background()

	var dependencies []uuid.UUID
	err := dm.db.Pool.QueryRow(ctx, `
		SELECT dependencies FROM distributed_tasks WHERE id = $1
	`, taskID).Scan(&dependencies)

	if err != nil {
		return nil, fmt.Errorf("failed to get task dependencies: %v", err)
	}

	return dependencies, nil
}

func (dm *DependencyManager) traverseDependencies(taskID uuid.UUID, visited map[uuid.UUID]bool, chain *[]uuid.UUID) error {
	if visited[taskID] {
		return nil // Already visited
	}

	visited[taskID] = true
	*chain = append(*chain, taskID)

	// Get dependencies
	dependencies, err := dm.getTaskDependencies(taskID)
	if err != nil {
		return err
	}

	// Recursively traverse dependencies
	for _, depID := range dependencies {
		if err := dm.traverseDependencies(depID, visited, chain); err != nil {
			return err
		}
	}

	return nil
}
