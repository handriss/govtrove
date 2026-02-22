CREATE SCHEMA IF NOT EXISTS pipeline;

ALTER TABLE public.ingestion_runs SET SCHEMA pipeline;
ALTER TABLE public.csv_cache_headers SET SCHEMA pipeline;
ALTER TABLE public.samgov_requests SET SCHEMA pipeline;
ALTER TABLE public.bulk_csv_log SET SCHEMA pipeline;
ALTER TABLE public.csv_download_log SET SCHEMA pipeline;
ALTER TABLE public.snap_csv SET SCHEMA pipeline;
ALTER TABLE public.snap_changes SET SCHEMA pipeline;
ALTER TABLE public.snap_disappearances SET SCHEMA pipeline;
ALTER TABLE public.snap_data_quality SET SCHEMA pipeline;

ALTER TABLE IF EXISTS public.snap_archived_csv SET SCHEMA pipeline;
ALTER TABLE IF EXISTS public.archived_csv_download_log SET SCHEMA pipeline;

CREATE TABLE pipeline.pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms INT,
    stats JSONB,
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX idx_pipeline_runs_pipeline_name ON pipeline.pipeline_runs(pipeline_name);
CREATE INDEX idx_pipeline_runs_status ON pipeline.pipeline_runs(status);
CREATE INDEX idx_pipeline_runs_started_at ON pipeline.pipeline_runs(started_at DESC);
