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
