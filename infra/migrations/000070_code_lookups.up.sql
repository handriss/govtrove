-- Usage log for the NAICS/PSC code finders. One row per /api/codes/*/match request.
CREATE TABLE IF NOT EXISTS code_lookups (
    id           BIGSERIAL PRIMARY KEY,
    code_type    TEXT        NOT NULL,          -- 'naics' | 'psc'
    description  TEXT        NOT NULL,          -- the business/service description the user typed
    result_count INTEGER     NOT NULL DEFAULT 0,
    top_code     TEXT,                          -- best match code (NULL if no matches)
    top_score    REAL,                          -- best match similarity 0..1
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_code_lookups_created_at ON code_lookups (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_code_lookups_code_type  ON code_lookups (code_type);
