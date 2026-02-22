-- Track HTTP cache headers for CSV downloads to enable conditional requests
CREATE TABLE csv_cache_headers (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    etag TEXT,
    last_modified TEXT,
    content_length BIGINT,
    last_checked_at TIMESTAMPTZ DEFAULT NOW(),
    last_downloaded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_csv_cache_headers_url ON csv_cache_headers(url);
