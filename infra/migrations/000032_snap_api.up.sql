CREATE TABLE pipeline.snap_api (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL,
    notice_id       VARCHAR(64) NOT NULL,
    raw_data        JSONB NOT NULL,
    content_hash    TEXT NOT NULL,
    snapshot_date   TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (run_id, notice_id)
);

CREATE INDEX idx_snap_api_run ON pipeline.snap_api(run_id);
CREATE INDEX idx_snap_api_notice ON pipeline.snap_api(notice_id);
