CREATE INDEX IF NOT EXISTS idx_opps_search ON opportunities USING gin (search_vector);
DROP INDEX IF EXISTS idx_opps_search_active;
