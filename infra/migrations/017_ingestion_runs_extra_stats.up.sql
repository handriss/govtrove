ALTER TABLE ingestion_runs
    ADD COLUMN IF NOT EXISTS records_skipped integer DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_db_count integer DEFAULT 0;
