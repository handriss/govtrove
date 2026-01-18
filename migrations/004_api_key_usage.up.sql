CREATE TABLE api_key_usage (
    id SERIAL PRIMARY KEY,
    request_timestamp TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    endpoint TEXT NOT NULL,
    method VARCHAR(10) NOT NULL,
    http_status_code INTEGER,
    response_time_ms INTEGER,
    request_params JSONB,
    response_size_bytes INTEGER,
    error_message TEXT,
    ingestion_run_id INTEGER REFERENCES ingestion_runs(id),
    success BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_api_key_usage_timestamp ON api_key_usage(request_timestamp DESC);
CREATE INDEX idx_api_key_usage_ingestion_run ON api_key_usage(ingestion_run_id);
CREATE INDEX idx_api_key_usage_success ON api_key_usage(success);
CREATE INDEX idx_api_key_usage_status_code ON api_key_usage(http_status_code);
