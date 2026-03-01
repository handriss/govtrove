ALTER TABLE pipeline.snap_reconcile_dq ADD COLUMN resolved_at TIMESTAMPTZ;
ALTER TABLE pipeline.snap_reconcile_dq ADD COLUMN resolution_note TEXT;
