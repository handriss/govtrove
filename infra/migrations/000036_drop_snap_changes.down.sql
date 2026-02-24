CREATE TABLE pipeline.snap_changes (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL REFERENCES pipeline.ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    source          TEXT NOT NULL,
    field_name      TEXT NOT NULL,
    old_value       TEXT,
    new_value       TEXT,
    detected_date   TIMESTAMPTZ NOT NULL,
    previous_date   TIMESTAMPTZ,
    change_type     TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_snap_changes_run_id ON pipeline.snap_changes(run_id);
CREATE INDEX idx_snap_changes_notice_id ON pipeline.snap_changes(notice_id);
