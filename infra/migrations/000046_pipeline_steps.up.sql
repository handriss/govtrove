CREATE TABLE pipeline.pipeline_steps (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id  UUID NOT NULL,
    step_name     TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'running',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    duration_ms   INT,
    stats         JSONB,
    error_message TEXT
);

CREATE INDEX idx_pipeline_steps_execution ON pipeline.pipeline_steps(execution_id);
CREATE INDEX idx_pipeline_steps_started ON pipeline.pipeline_steps(started_at DESC);
