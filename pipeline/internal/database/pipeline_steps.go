package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PipelineStep struct {
	ID           uuid.UUID
	ExecutionID  uuid.UUID
	StepName     string
	Status       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	DurationMs   *int
	Stats        map[string]any
	ErrorMessage *string
}

func (db *DB) CreatePipelineStep(ctx context.Context, executionID uuid.UUID, stepName string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.pool.QueryRow(ctx, `
		INSERT INTO pipeline.pipeline_steps (execution_id, step_name)
		VALUES ($1, $2)
		RETURNING id
	`, executionID, stepName).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create pipeline step: %w", err)
	}
	return id, nil
}

func (db *DB) CompletePipelineStep(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error {
	var statsStr *string
	if stats != nil {
		b, err := json.Marshal(stats)
		if err != nil {
			return fmt.Errorf("marshal stats: %w", err)
		}
		s := string(b)
		statsStr = &s
	}

	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.pipeline_steps
		SET status = 'completed',
		    completed_at = now(),
		    duration_ms = $2,
		    stats = $3
		WHERE id = $1
	`, id, durationMs, statsStr)
	if err != nil {
		return fmt.Errorf("complete pipeline step: %w", err)
	}
	return nil
}

func (db *DB) FailPipelineStep(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.pipeline_steps
		SET status = 'failed',
		    completed_at = now(),
		    duration_ms = $2,
		    error_message = $3
		WHERE id = $1
	`, id, durationMs, errMsg)
	if err != nil {
		return fmt.Errorf("fail pipeline step: %w", err)
	}
	return nil
}
