DROP INDEX IF EXISTS idx_search_events_user_id;
ALTER TABLE search_events DROP COLUMN user_id;
ALTER TABLE search_events ADD COLUMN session_id VARCHAR(64);
ALTER TABLE search_events ADD COLUMN result_position INTEGER;
CREATE INDEX idx_search_events_session_id ON search_events(session_id);
