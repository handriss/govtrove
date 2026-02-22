-- Migration 023: Snapshot-based ingestion schema
-- Drops legacy tables, renames archive logs, creates snapshot pipeline tables.

-- 1. Drop legacy tables (manually renamed, stale data, confirmed disposable)
DROP TABLE IF EXISTS opportunity_contacts CASCADE;
DROP TABLE IF EXISTS opportunity_resources CASCADE;
DROP TABLE IF EXISTS opportunities_v1 CASCADE;
DROP TABLE IF EXISTS ingestion_runs_v1 CASCADE;

-- 2. Rename existing archive log tables (preserve data, csvarchive/archivedcsv use them)
ALTER TABLE csv_download_log RENAME TO csv_s3_archive_log;
ALTER INDEX idx_csv_download_log_checked_at RENAME TO idx_csv_s3_archive_log_checked_at;
ALTER INDEX idx_csv_download_log_sha256 RENAME TO idx_csv_s3_archive_log_sha256;

ALTER TABLE archived_csv_download_log RENAME TO archived_csv_s3_archive_log;
ALTER INDEX idx_archived_csv_dl_fy_result RENAME TO idx_archived_csv_s3_archive_fy_result;

-- 3. Ingestion runs (UUID primary key, used by all jobs)
CREATE TABLE ingestion_runs (
    run_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type        TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'running',
    records_fetched   INT DEFAULT 0,
    records_inserted  INT DEFAULT 0,
    records_updated   INT DEFAULT 0,
    records_failed    INT DEFAULT 0,
    records_skipped   INT DEFAULT 0,
    total_db_count    INT DEFAULT 0,
    error_message   TEXT,
    duration_ms     INT
);

CREATE INDEX idx_ingestion_runs_status ON ingestion_runs(status);
CREATE INDEX idx_ingestion_runs_job_type ON ingestion_runs(job_type, started_at DESC);

-- 4. CSV download log (per-run download tracking for ingestion pipeline)
CREATE TABLE csv_download_log (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID REFERENCES ingestion_runs(run_id),
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

CREATE INDEX idx_csv_download_log_run_id ON csv_download_log(run_id);

-- 5. Archived CSV download log (same structure, for archived CSV ingestion)
CREATE TABLE archived_csv_download_log (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID REFERENCES ingestion_runs(run_id),
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

CREATE INDEX idx_archived_csv_download_log_run_id ON archived_csv_download_log(run_id);

-- 6. snap_csv — one row per notice per run, all text columns
CREATE TABLE snap_csv (
    id              BIGSERIAL PRIMARY KEY,

    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    title           TEXT,
    type            TEXT,
    base_type       TEXT,
    posted_date     TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date    TEXT,
    archive_type    TEXT,
    set_aside_code  TEXT,
    naics_code      TEXT,
    classification_code TEXT,
    active          BOOLEAN,

    department      TEXT,
    sub_tier        TEXT,
    office          TEXT,
    cgac            TEXT,
    fpds_code       TEXT,
    aac_code        TEXT,

    award_number    TEXT,
    award_date      TEXT,
    award_amount    NUMERIC(15,2),

    raw_data        JSONB NOT NULL,
    content_hash    TEXT NOT NULL,

    run_id          UUID NOT NULL REFERENCES ingestion_runs(run_id),
    snapshot_date   TIMESTAMPTZ NOT NULL,
    download_id     BIGINT REFERENCES csv_download_log(id),
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(notice_id, run_id)
);

CREATE INDEX idx_snap_csv_notice_id ON snap_csv(notice_id);
CREATE INDEX idx_snap_csv_download_id ON snap_csv(download_id);
CREATE INDEX idx_snap_csv_snapshot_date ON snap_csv(snapshot_date);
CREATE INDEX idx_snap_csv_run_id ON snap_csv(run_id);
CREATE INDEX idx_snap_csv_hash ON snap_csv(notice_id, run_id, content_hash);
CREATE INDEX idx_snap_csv_type ON snap_csv(type, snapshot_date);
CREATE INDEX idx_snap_csv_sol_num ON snap_csv(solicitation_number) WHERE solicitation_number IS NOT NULL;

-- 7. snap_archived_csv — same structure, different download_id FK
CREATE TABLE snap_archived_csv (
    id              BIGSERIAL PRIMARY KEY,

    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    title           TEXT,
    type            TEXT,
    base_type       TEXT,
    posted_date     TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date    TEXT,
    archive_type    TEXT,
    set_aside_code  TEXT,
    naics_code      TEXT,
    classification_code TEXT,
    active          BOOLEAN,

    department      TEXT,
    sub_tier        TEXT,
    office          TEXT,
    cgac            TEXT,
    fpds_code       TEXT,
    aac_code        TEXT,

    award_number    TEXT,
    award_date      TEXT,
    award_amount    NUMERIC(15,2),

    raw_data        JSONB NOT NULL,
    content_hash    TEXT NOT NULL,

    run_id          UUID NOT NULL REFERENCES ingestion_runs(run_id),
    snapshot_date   TIMESTAMPTZ NOT NULL,
    download_id     BIGINT REFERENCES archived_csv_download_log(id),
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(notice_id, run_id)
);

CREATE INDEX idx_snap_arch_notice_id ON snap_archived_csv(notice_id);
CREATE INDEX idx_snap_arch_download_id ON snap_archived_csv(download_id);
CREATE INDEX idx_snap_arch_snapshot_date ON snap_archived_csv(snapshot_date);
CREATE INDEX idx_snap_arch_run_id ON snap_archived_csv(run_id);
CREATE INDEX idx_snap_arch_hash ON snap_archived_csv(notice_id, run_id, content_hash);
CREATE INDEX idx_snap_arch_sol_num ON snap_archived_csv(solicitation_number) WHERE solicitation_number IS NOT NULL;

-- 8. snap_changes — field-level change log
CREATE TABLE snap_changes (
    id              BIGSERIAL PRIMARY KEY,

    run_id          UUID NOT NULL REFERENCES ingestion_runs(run_id),
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

CREATE INDEX idx_snap_changes_notice_id ON snap_changes(notice_id);
CREATE INDEX idx_snap_changes_run_id ON snap_changes(run_id);
CREATE INDEX idx_snap_changes_detected ON snap_changes(detected_date);
CREATE INDEX idx_snap_changes_sol_num ON snap_changes(solicitation_number) WHERE solicitation_number IS NOT NULL;
CREATE INDEX idx_snap_changes_field ON snap_changes(field_name, detected_date);
CREATE INDEX idx_snap_changes_type ON snap_changes(change_type, detected_date);
CREATE INDEX idx_snap_changes_notice_date ON snap_changes(notice_id, detected_date);

-- 9. snap_disappearances — tracks records vanishing from active CSV
CREATE TABLE snap_disappearances (
    id              BIGSERIAL PRIMARY KEY,

    run_id          UUID NOT NULL REFERENCES ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,

    last_seen_date  TIMESTAMPTZ NOT NULL,
    disappeared_date TIMESTAMPTZ NOT NULL,

    last_type       TEXT,
    last_archive_type TEXT,
    last_archive_date TEXT,

    resolution      TEXT,
    resolution_date TIMESTAMPTZ,
    resolution_source TEXT,
    reappeared_date TIMESTAMPTZ,
    notes           TEXT,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snap_disapp_notice ON snap_disappearances(notice_id);
CREATE INDEX idx_snap_disapp_run_id ON snap_disappearances(run_id);
CREATE INDEX idx_snap_disapp_date ON snap_disappearances(disappeared_date);
CREATE INDEX idx_snap_disapp_unresolved ON snap_disappearances(resolution) WHERE resolution IS NULL;
CREATE INDEX idx_snap_disapp_sol_num ON snap_disappearances(solicitation_number) WHERE solicitation_number IS NOT NULL;

-- 10. snap_data_quality — future use, not populated by Go code yet
CREATE TABLE snap_data_quality (
    id              BIGSERIAL PRIMARY KEY,

    run_id          UUID NOT NULL REFERENCES ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    snapshot_date   TIMESTAMPTZ NOT NULL,
    source          TEXT NOT NULL,

    issue_type      TEXT NOT NULL,
    field_name      TEXT,
    field_value     TEXT,
    description     TEXT,

    resolved        BOOLEAN DEFAULT false,
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT,

    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_snap_dq_notice ON snap_data_quality(notice_id);
CREATE INDEX idx_snap_dq_run_id ON snap_data_quality(run_id);
CREATE INDEX idx_snap_dq_date ON snap_data_quality(snapshot_date);
CREATE INDEX idx_snap_dq_type ON snap_data_quality(issue_type);
CREATE INDEX idx_snap_dq_unresolved ON snap_data_quality(resolved) WHERE resolved = false;
