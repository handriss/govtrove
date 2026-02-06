ALTER TABLE ingestion_runs ADD COLUMN ingestion_mode VARCHAR(32) DEFAULT 'full';
ALTER TABLE ingestion_runs ADD COLUMN lookback_days INTEGER;
