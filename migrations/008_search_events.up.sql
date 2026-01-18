CREATE TABLE search_events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(32) NOT NULL,
    session_id VARCHAR(64),

    query TEXT,
    filters JSONB,
    sort_by VARCHAR(32),
    page INTEGER,

    total_results INTEGER,
    result_position INTEGER,
    opportunity_id INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    user_agent TEXT,
    referer TEXT
);

CREATE INDEX idx_search_events_created_at ON search_events(created_at DESC);
CREATE INDEX idx_search_events_event_type ON search_events(event_type);
CREATE INDEX idx_search_events_session_id ON search_events(session_id);
CREATE INDEX idx_search_events_query ON search_events USING GIN(to_tsvector('english', query));
