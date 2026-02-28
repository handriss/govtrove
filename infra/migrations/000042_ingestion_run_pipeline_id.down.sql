DROP INDEX IF EXISTS pipeline.idx_ingestion_runs_pipeline_run;

ALTER TABLE pipeline.ingestion_runs
    DROP COLUMN IF EXISTS pipeline_run_id;
