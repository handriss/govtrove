DROP INDEX IF EXISTS idx_search_events_utm_campaign;
ALTER TABLE search_events DROP COLUMN IF EXISTS utm_campaign;

DROP TABLE IF EXISTS utm_visits;
