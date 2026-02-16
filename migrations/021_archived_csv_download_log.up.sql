CREATE TABLE archived_csv_download_log (
    id                      SERIAL PRIMARY KEY,
    fiscal_year             INTEGER NOT NULL,
    checked_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result                  TEXT NOT NULL,
    http_status             INTEGER,
    etag                    TEXT,
    last_modified           TEXT,
    content_length          BIGINT,
    file_size_bytes         BIGINT,
    compressed_size_bytes   BIGINT,
    row_count               INTEGER,
    sha256_hash             TEXT,
    s3_key                  TEXT,
    download_duration_ms    INTEGER,
    compression_duration_ms INTEGER,
    upload_duration_ms      INTEGER,
    error_message           TEXT
);

CREATE INDEX idx_archived_csv_dl_fy_result ON archived_csv_download_log(fiscal_year, result, checked_at DESC);
