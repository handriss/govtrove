-- Migration 041: Consolidate csv_download_log into bulk_csv_log
--
-- bulk_csv_log already tracks every CSV download/check. csv_download_log and
-- archived_csv_download_log duplicate this for the ingest step. Unifying them
-- gives a single audit trail per file across the entire pipeline.

-- 1. Add new columns to bulk_csv_log
ALTER TABLE pipeline.bulk_csv_log
    ADD COLUMN IF NOT EXISTS pipeline_run_id UUID REFERENCES pipeline.pipeline_runs(id),
    ADD COLUMN IF NOT EXISTS ingestion_run_id UUID,
    ADD COLUMN IF NOT EXISTS record_count INT,
    ADD COLUMN IF NOT EXISTS status TEXT;

CREATE INDEX IF NOT EXISTS idx_bulk_csv_log_pipeline_run
    ON pipeline.bulk_csv_log(pipeline_run_id) WHERE pipeline_run_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_bulk_csv_log_s3_key
    ON pipeline.bulk_csv_log(s3_key) WHERE s3_key IS NOT NULL AND s3_key != '';

-- 2. Drop FK constraints on snap_csv.download_id and snap_archived_csv.download_id
ALTER TABLE pipeline.snap_csv DROP CONSTRAINT IF EXISTS snap_csv_download_id_fkey;
ALTER TABLE pipeline.snap_archived_csv DROP CONSTRAINT IF EXISTS snap_archived_csv_download_id_fkey;

-- 3. Drop the old tables
DROP TABLE IF EXISTS pipeline.csv_download_log CASCADE;
DROP TABLE IF EXISTS pipeline.archived_csv_download_log CASCADE;
