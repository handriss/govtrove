CREATE TABLE pipeline.snap_reconcile_dq (
    id              BIGSERIAL PRIMARY KEY,
    csv_run_id      UUID REFERENCES pipeline.ingestion_runs(run_id),
    api_run_id      UUID REFERENCES pipeline.ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    snapshot_date   TIMESTAMPTZ NOT NULL,
    issue_type      TEXT NOT NULL,
    field_name      TEXT NOT NULL,
    csv_value       TEXT,
    api_value       TEXT,
    resolved        BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snap_reconcile_dq_notice ON pipeline.snap_reconcile_dq(notice_id);
CREATE INDEX idx_snap_reconcile_dq_csv_run ON pipeline.snap_reconcile_dq(csv_run_id);
CREATE INDEX idx_snap_reconcile_dq_api_run ON pipeline.snap_reconcile_dq(api_run_id);
