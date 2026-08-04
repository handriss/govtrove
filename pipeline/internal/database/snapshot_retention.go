package database

import (
	"context"
	"fmt"
)

// DeleteOldSnapshots trims the append-only debug/audit snapshot tables so they don't grow
// unbounded — they carry full raw ingest payloads and reconcile discrepancies the live app
// never reads, and were the driver behind the Neon storage/bill spikes. It runs at the end
// of each pipeline cycle (in reconcile), after fresh snapshots have been written, so the
// most recent snapshot is always well within the retention window.
//
// snapDays applies to the raw CSV/API snapshots; dqDays to the reconcile / data-quality logs.
// Returns total rows deleted.
func (db *DB) DeleteOldSnapshots(ctx context.Context, snapDays, dqDays int) (int64, error) {
	targets := []struct {
		table string
		days  int
	}{
		{"pipeline.snap_csv", snapDays},
		{"pipeline.snap_api", snapDays},
		{"pipeline.snap_reconcile_dq", dqDays},
		{"pipeline.snap_data_quality", dqDays},
	}

	var total int64
	for _, t := range targets {
		tag, err := db.pool.Exec(ctx,
			fmt.Sprintf(`DELETE FROM %s WHERE snapshot_date < NOW() - INTERVAL '1 day' * $1`, t.table),
			t.days)
		if err != nil {
			return total, fmt.Errorf("pruning %s: %w", t.table, err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
}
