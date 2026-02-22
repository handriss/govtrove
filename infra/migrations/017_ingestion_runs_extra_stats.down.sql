ALTER TABLE ingestion_runs
    DROP COLUMN IF EXISTS records_skipped,
    DROP COLUMN IF EXISTS total_db_count;
