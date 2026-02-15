CREATE TABLE api_update_probe (
    id SERIAL PRIMARY KEY,
    probed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_records INT,
    posted_from TEXT,
    sample_notice_id TEXT,
    sample_posted_date TEXT,
    http_status INT,
    response_time_ms INT,
    error_message TEXT
);
CREATE INDEX idx_api_update_probe_probed_at ON api_update_probe(probed_at DESC);

CREATE TABLE api_fetch_log (
    id SERIAL PRIMARY KEY,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result TEXT NOT NULL,
    posted_from TEXT,
    total_records INT,
    records_fetched INT,
    pages_fetched INT,
    file_size_bytes BIGINT,
    compressed_size_bytes BIGINT,
    sha256_hash TEXT,
    s3_key TEXT,
    fetch_duration_ms INT,
    compression_duration_ms INT,
    upload_duration_ms INT,
    error_message TEXT
);
CREATE INDEX idx_api_fetch_log_fetched_at ON api_fetch_log(fetched_at DESC);
CREATE INDEX idx_api_fetch_log_sha256 ON api_fetch_log(sha256_hash);
