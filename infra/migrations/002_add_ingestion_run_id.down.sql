DROP INDEX IF EXISTS idx_opportunities_ingestion_run;

ALTER TABLE opportunities DROP COLUMN IF EXISTS ingestion_run_id;
