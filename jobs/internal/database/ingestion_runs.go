package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RunStats struct {
	Fetched    int
	Inserted   int
	Updated    int
	Failed     int
	Skipped    int
	TotalDB    int
	DurationMs int
}

func (db *DB) CreateIngestionRun(ctx context.Context, jobType string) (uuid.UUID, error) {
	var runID uuid.UUID
	err := db.pool.QueryRow(ctx,
		`INSERT INTO pipeline.ingestion_runs (job_type, status) VALUES ($1, 'running') RETURNING run_id`,
		jobType,
	).Scan(&runID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create ingestion run: %w", err)
	}
	return runID, nil
}

func (db *DB) CompleteIngestionRun(ctx context.Context, runID uuid.UUID, s RunStats) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.ingestion_runs
		SET status = 'completed',
		    completed_at = NOW(),
		    records_fetched = $2,
		    records_inserted = $3,
		    records_updated = $4,
		    records_failed = $5,
		    records_skipped = $6,
		    total_db_count = $7,
		    duration_ms = $8
		WHERE run_id = $1
	`, runID, s.Fetched, s.Inserted, s.Updated, s.Failed, s.Skipped, s.TotalDB, s.DurationMs)
	if err != nil {
		return fmt.Errorf("failed to complete ingestion run: %w", err)
	}
	return nil
}

func (db *DB) FailIngestionRun(ctx context.Context, runID uuid.UUID, errMsg string, durationMs int) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE pipeline.ingestion_runs
		SET status = 'failed',
		    completed_at = NOW(),
		    error_message = $2,
		    duration_ms = $3
		WHERE run_id = $1
	`, runID, errMsg, durationMs)
	if err != nil {
		return fmt.Errorf("failed to mark ingestion run as failed: %w", err)
	}
	return nil
}

func (db *DB) GetLastCompletedRun(ctx context.Context, jobType string) (uuid.UUID, time.Time, error) {
	var runID uuid.UUID
	var snapshotDate time.Time
	err := db.pool.QueryRow(ctx, `
		SELECT run_id, started_at
		FROM pipeline.ingestion_runs
		WHERE job_type = $1 AND status = 'completed'
		ORDER BY started_at DESC
		LIMIT 1
	`, jobType).Scan(&runID, &snapshotDate)
	if err == pgx.ErrNoRows {
		return uuid.Nil, time.Time{}, nil
	}
	if err != nil {
		return uuid.Nil, time.Time{}, fmt.Errorf("failed to get last completed run: %w", err)
	}
	return runID, snapshotDate, nil
}
