ALTER TABLE search_events DROP CONSTRAINT IF EXISTS search_events_user_id_fkey;
ALTER TABLE search_events ADD CONSTRAINT search_events_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);
