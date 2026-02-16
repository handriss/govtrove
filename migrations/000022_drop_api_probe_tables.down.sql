-- Recreate tables if needed (from migration 020)
CREATE TABLE IF NOT EXISTS api_update_probe (
    id              SERIAL PRIMARY KEY,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_records   INTEGER,
    latest_modified TIMESTAMPTZ,
    latest_posted   TIMESTAMPTZ,
    response_time_ms INTEGER,
    error_message   TEXT
);

CREATE TABLE IF NOT EXISTS api_fetch_log (
    id               SERIAL PRIMARY KEY,
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result           TEXT NOT NULL,
    notice_id        TEXT,
    sol_number       TEXT,
    posted_date      DATE,
    modified_date    TIMESTAMPTZ,
    response_time_ms INTEGER,
    s3_key           TEXT,
    error_message    TEXT
);

CREATE INDEX IF NOT EXISTS idx_api_probe_checked ON api_update_probe(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_fetch_result ON api_fetch_log(result, fetched_at DESC);
