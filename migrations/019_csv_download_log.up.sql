CREATE TABLE csv_download_log (
    id SERIAL PRIMARY KEY,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result TEXT NOT NULL,  -- 'cache_hit', 'hash_match', 'new_file', 'error'
    http_status INT,
    etag TEXT,
    last_modified TEXT,
    file_size_bytes BIGINT,
    compressed_size_bytes BIGINT,
    row_count INT,
    sha256_hash TEXT,
    s3_key TEXT,
    download_duration_ms INT,
    compression_duration_ms INT,
    upload_duration_ms INT,
    error_message TEXT
);
CREATE INDEX idx_csv_download_log_checked_at ON csv_download_log(checked_at DESC);
CREATE INDEX idx_csv_download_log_sha256 ON csv_download_log(sha256_hash);
