-- Restore old tables (without data)
CREATE TABLE IF NOT EXISTS bulk_csv_active_log (
    id                      SERIAL PRIMARY KEY,
    checked_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result                  TEXT,
    http_status             INT,
    etag                    TEXT,
    last_modified           TEXT,
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

CREATE TABLE IF NOT EXISTS bulk_csv_archived_log (
    id                      SERIAL PRIMARY KEY,
    fiscal_year             INTEGER NOT NULL,
    checked_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result                  TEXT,
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

DROP TABLE IF EXISTS bulk_csv_log;
