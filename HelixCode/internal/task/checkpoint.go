package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dev.helix.code/internal/database"
	"github.com/google/uuid"
)

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(db *database.Database) *CheckpointManager {
	return &CheckpointManager{
		db: db,
	}
}

// CreateCheckpoint creates a checkpoint for a task
func (cm *CheckpointManager) CreateCheckpoint(taskID uuid.UUID, checkpointName string, checkpointData map[string]interface{}) error {
	ctx := context.Background()

	// Convert checkpoint data to JSON
	checkpointDataJSON, err := json.Marshal(checkpointData)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint data: %v", err)
	}

	// Get the current worker ID (this would come from the task context)
	workerID := uuid.New() // In real implementation, this would be the actual worker ID

	// Insert checkpoint into database
	_, err = cm.db.Pool.Exec(ctx, `
		INSERT INTO task_checkpoints (
			id, task_id, checkpoint_name, checkpoint_data, worker_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`,
		uuid.New(), taskID, checkpointName, checkpointDataJSON, workerID, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %v", err)
	}

	return nil
}

// GetCheckpoints returns all checkpoints for a task
func (cm *CheckpointManager) GetCheckpoints(taskID uuid.UUID) ([]Checkpoint, error) {
	ctx := context.Background()

	rows, err := cm.db.Pool.Query(ctx, `
		SELECT id, checkpoint_name, checkpoint_data, worker_id, created_at
		FROM task_checkpoints
		WHERE task_id = $1
		ORDER BY created_at DESC
	`, taskID)

	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %v", err)
	}
	defer rows.Close()

	var checkpoints []Checkpoint
	for rows.Next() {
		var checkpoint Checkpoint
		var checkpointDataJSON []byte

		err := rows.Scan(
			&checkpoint.ID,
			&checkpoint.CheckpointName,
			&checkpointDataJSON,
			&checkpoint.WorkerID,
			&checkpoint.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %v", err)
		}

		// Parse checkpoint data
		if err := json.Unmarshal(checkpointDataJSON, &checkpoint.CheckpointData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal checkpoint data: %v", err)
		}

		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

// GetLatestCheckpoint returns the latest checkpoint for a task
func (cm *CheckpointManager) GetLatestCheckpoint(taskID uuid.UUID) (*Checkpoint, error) {
	ctx := context.Background()

	var checkpoint Checkpoint
	var checkpointDataJSON []byte

	err := cm.db.Pool.QueryRow(ctx, `
		SELECT id, checkpoint_name, checkpoint_data, worker_id, created_at
		FROM task_checkpoints
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, taskID).Scan(
		&checkpoint.ID,
		&checkpoint.CheckpointName,
		&checkpointDataJSON,
		&checkpoint.WorkerID,
		&checkpoint.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get latest checkpoint: %v", err)
	}

	// Parse checkpoint data
	if err := json.Unmarshal(checkpointDataJSON, &checkpoint.CheckpointData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint data: %v", err)
	}

	return &checkpoint, nil
}

// DeleteCheckpoint deletes a specific checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(checkpointID uuid.UUID) error {
	ctx := context.Background()

	_, err := cm.db.Pool.Exec(ctx, `
		DELETE FROM task_checkpoints
		WHERE id = $1
	`, checkpointID)

	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %v", err)
	}

	return nil
}

// DeleteAllCheckpoints deletes all checkpoints for a task
func (cm *CheckpointManager) DeleteAllCheckpoints(taskID uuid.UUID) error {
	ctx := context.Background()

	_, err := cm.db.Pool.Exec(ctx, `
		DELETE FROM task_checkpoints
		WHERE task_id = $1
	`, taskID)

	if err != nil {
		return fmt.Errorf("failed to delete checkpoints: %v", err)
	}

	return nil
}

// Checkpoint represents a task checkpoint
type Checkpoint struct {
	ID             uuid.UUID              `json:"id"`
	CheckpointName string                 `json:"checkpoint_name"`
	CheckpointData map[string]interface{} `json:"checkpoint_data"`
	WorkerID       uuid.UUID              `json:"worker_id"`
	CreatedAt      time.Time              `json:"created_at"`
}
