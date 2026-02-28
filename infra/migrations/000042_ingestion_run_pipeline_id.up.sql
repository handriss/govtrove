ALTER TABLE pipeline.ingestion_runs
    ADD COLUMN IF NOT EXISTS pipeline_run_id UUID REFERENCES pipeline.pipeline_runs(id);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_pipeline_run
    ON pipeline.ingestion_runs(pipeline_run_id) WHERE pipeline_run_id IS NOT NULL;
