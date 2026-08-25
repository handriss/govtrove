-- Account deletion (scripts/gdpr.sh) failed with a foreign-key violation for any
-- user who had ever searched, against a published 24-hour deletion promise.
-- SET NULL rather than CASCADE: the event survives de-identified, so aggregate
-- search analytics stay intact while the row stops pointing at a deleted person.
ALTER TABLE search_events DROP CONSTRAINT IF EXISTS search_events_user_id_fkey;
ALTER TABLE search_events ADD CONSTRAINT search_events_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
