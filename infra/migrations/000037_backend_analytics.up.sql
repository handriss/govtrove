ALTER TABLE search_events ADD COLUMN user_id INTEGER REFERENCES users(id);
ALTER TABLE search_events DROP COLUMN session_id;
ALTER TABLE search_events DROP COLUMN result_position;
DROP INDEX IF EXISTS idx_search_events_session_id;
CREATE INDEX idx_search_events_user_id ON search_events(user_id) WHERE user_id IS NOT NULL;
