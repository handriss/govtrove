CREATE TABLE bulk_csv_log (
    id                      SERIAL PRIMARY KEY,
    source                  TEXT NOT NULL,
    checked_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result                  TEXT NOT NULL,
    http_status             INT,
    etag                    TEXT,
    last_modified           TEXT,
    content_length          BIGINT,
    file_size_bytes         BIGINT,
    compressed_size_bytes   BIGINT,
    row_count               INT,
    sha256_hash             TEXT,
    s3_key                  TEXT,
    download_duration_ms    INT,
    compression_duration_ms INT,
    upload_duration_ms      INT,
    error_message           TEXT
);

CREATE INDEX idx_bulk_csv_log_source_checked ON bulk_csv_log(source, checked_at DESC);
CREATE INDEX idx_bulk_csv_log_source_hash ON bulk_csv_log(source, sha256_hash) WHERE sha256_hash IS NOT NULL;

DROP TABLE IF EXISTS bulk_csv_active_log;
DROP TABLE IF EXISTS bulk_csv_archived_log;
