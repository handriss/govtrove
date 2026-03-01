package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SnapAPIRow struct {
	NoticeID    string
	RawData     json.RawMessage
	ContentHash string
}

func (db *DB) BulkInsertSnapAPI(ctx context.Context, runID uuid.UUID, snapshotDate time.Time, rows []SnapAPIRow) (int64, error) {
	columns := []string{
		"run_id", "notice_id", "raw_data", "content_hash", "snapshot_date",
	}

	copyCount, err := db.pool.CopyFrom(
		ctx,
		pgx.Identifier{"pipeline", "snap_api"},
		columns,
		&snapAPICopySource{rows: rows, runID: runID, snapshotDate: snapshotDate},
	)
	if err != nil {
		return 0, fmt.Errorf("bulk insert snap_api failed: %w", err)
	}
	return copyCount, nil
}

type SnapAPIRawRow struct {
	NoticeID string
	RawData  json.RawMessage
}

func (db *DB) GetSnapAPIRawData(ctx context.Context, runID uuid.UUID) ([]SnapAPIRawRow, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT notice_id, raw_data FROM pipeline.snap_api WHERE run_id = $1`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("query snap_api raw_data: %w", err)
	}
	defer rows.Close()

	var result []SnapAPIRawRow
	for rows.Next() {
		var r SnapAPIRawRow
		if err := rows.Scan(&r.NoticeID, &r.RawData); err != nil {
			return nil, fmt.Errorf("scan snap_api row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

type snapAPICopySource struct {
	rows         []SnapAPIRow
	runID        uuid.UUID
	snapshotDate time.Time
	idx          int
}

func (s *snapAPICopySource) Next() bool {
	s.idx++
	return s.idx <= len(s.rows)
}

func (s *snapAPICopySource) Values() ([]interface{}, error) {
	row := s.rows[s.idx-1]
	return []interface{}{
		s.runID,
		row.NoticeID,
		row.RawData,
		row.ContentHash,
		s.snapshotDate,
	}, nil
}

func (s *snapAPICopySource) Err() error { return nil }
