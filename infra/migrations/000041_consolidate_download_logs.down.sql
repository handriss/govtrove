-- Reverse migration 041: recreate csv_download_log tables, remove new columns

CREATE TABLE IF NOT EXISTS pipeline.csv_download_log (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID,
    url             TEXT,
    download_date   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          TEXT NOT NULL DEFAULT 'downloading',
    etag            TEXT,
    last_modified   TEXT,
    file_size_bytes BIGINT,
    record_count    INT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline.archived_csv_download_log (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID,
    url             TEXT,
    download_date   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          TEXT NOT NULL DEFAULT 'downloading',
    etag            TEXT,
    last_modified   TEXT,
    file_size_bytes BIGINT,
    record_count    INT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DROP INDEX IF EXISTS pipeline.idx_bulk_csv_log_pipeline_run;
DROP INDEX IF EXISTS pipeline.idx_bulk_csv_log_s3_key;

ALTER TABLE pipeline.bulk_csv_log
    DROP COLUMN IF EXISTS pipeline_run_id,
    DROP COLUMN IF EXISTS ingestion_run_id,
    DROP COLUMN IF EXISTS record_count,
    DROP COLUMN IF EXISTS status;
