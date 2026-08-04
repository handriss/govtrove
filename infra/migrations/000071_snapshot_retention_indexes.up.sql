-- Indexes to make snapshot-retention DELETEs (and one-time prunes) fast — these tables
-- have no snapshot_date index, so filtering by it was a full seq scan over GBs of raw_data.
CREATE INDEX IF NOT EXISTS idx_snap_csv_snapshot_date          ON pipeline.snap_csv (snapshot_date);
CREATE INDEX IF NOT EXISTS idx_snap_api_snapshot_date          ON pipeline.snap_api (snapshot_date);
CREATE INDEX IF NOT EXISTS idx_snap_reconcile_dq_snapshot_date ON pipeline.snap_reconcile_dq (snapshot_date);
CREATE INDEX IF NOT EXISTS idx_snap_data_quality_snapshot_date ON pipeline.snap_data_quality (snapshot_date);
