CREATE TABLE user_updates (
    id UUID PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    update_type VARCHAR NOT NULL,
    source_id INTEGER,
    opportunity_ids INTEGER[],
    summary TEXT NOT NULL,
    details JSONB,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_user_updates_feed ON user_updates (user_id, is_read, created_at DESC);
CREATE INDEX idx_user_updates_source ON user_updates (source_id);
