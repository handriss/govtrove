package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PipelineRun struct {
	ID           uuid.UUID
	PipelineName string
	Status       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	DurationMs   *int
	Stats        map[string]any
	ErrorMessage *string
	Metadata     map[string]any
}

func (db *DB) CreatePipelineRun(ctx context.Context, id uuid.UUID, pipelineName string, metadata map[string]any) (uuid.UUID, error) {
	var metaStr *string
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshal metadata: %w", err)
		}
		s := string(b)
		metaStr = &s
	}

	var resultID uuid.UUID

	if id != uuid.Nil {
		// Caller-specified ID — use ON CONFLICT for idempotency
		err := db.pool.QueryRow(ctx, `
			INSERT INTO pipeline.pipeline_runs (id, pipeline_name, metadata)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO NOTHING
			RETURNING id
		`, id, pipelineName, metaStr).Scan(&resultID)
		if err == pgx.ErrNoRows {
			// Row already existed — return the caller's ID
			return id, nil
		}
		if err != nil {
			return uuid.Nil, fmt.Errorf("create pipeline run: %w", err)
		}
		return resultID, nil
	}

	// No ID specified — let Postgres generate one
	err := db.pool.QueryRow(ctx, `
		INSERT INTO pipeline.pipeline_runs (pipeline_name, metadata)
		VALUES ($1, $2)
		RETURNING id
	`, pipelineName, metaStr).Scan(&resultID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create pipeline run: %w", err)
	}
	return resultID, nil
}

func (db *DB) GetPipelineRun(ctx context.Context, id uuid.UUID) (*PipelineRun, error) {
	var r PipelineRun
	var statsJSON, metaJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, pipeline_name, status, started_at, completed_at, duration_ms, stats, error_message, metadata
		FROM pipeline.pipeline_runs
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.PipelineName, &r.Status, &r.StartedAt, &r.CompletedAt,
		&r.DurationMs, &statsJSON, &r.ErrorMessage, &metaJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline run: %w", err)
	}

	if statsJSON != nil {
		json.Unmarshal(statsJSON, &r.Stats)
	}
	if metaJSON != nil {
		json.Unmarshal(metaJSON, &r.Metadata)
	}

	return &r, nil
}

func (db *DB) CompletePipelineRun(ctx context.Context, id uuid.UUID, stats map[string]any, durationMs int) error {
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
		UPDATE pipeline.pipeline_runs
		SET status = 'completed',
		    completed_at = now(),
		    duration_ms = $2,
		    stats = $3
		WHERE id = $1
	`, id, durationMs, statsStr)
	if err != nil {
		return fmt.Errorf("complete pipeline run: %w", err)
	}
	return nil
}

func (db *DB) FailPipelineRun(ctx context.Context, id uuid.UUID, errMsg string, durationMs int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.pipeline_runs
		SET status = 'failed',
		    completed_at = now(),
		    duration_ms = $2,
		    error_message = $3
		WHERE id = $1
	`, id, durationMs, errMsg)
	if err != nil {
		return fmt.Errorf("fail pipeline run: %w", err)
	}
	return nil
}

func (db *DB) GetLatestPipelineRun(ctx context.Context, pipelineName string) (*PipelineRun, error) {
	var r PipelineRun
	var statsJSON, metaJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, pipeline_name, status, started_at, completed_at, duration_ms, stats, error_message, metadata
		FROM pipeline.pipeline_runs
		WHERE pipeline_name = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, pipelineName).Scan(
		&r.ID, &r.PipelineName, &r.Status, &r.StartedAt, &r.CompletedAt,
		&r.DurationMs, &statsJSON, &r.ErrorMessage, &metaJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest pipeline run: %w", err)
	}

	if statsJSON != nil {
		json.Unmarshal(statsJSON, &r.Stats)
	}
	if metaJSON != nil {
		json.Unmarshal(metaJSON, &r.Metadata)
	}

	return &r, nil
}
