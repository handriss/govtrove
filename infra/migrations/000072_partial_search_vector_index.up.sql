-- Partial GIN index on search_vector, scoped to active+latest rows only.
-- Every full-text consumer (API search, MCP, generate-alerts) hardcodes
-- "active = true AND is_latest = true", so this covers 100% of FTS queries while
-- indexing ~66k rows instead of all ~288k versions. A broad term like 'services'
-- dropped from ~65s to ~120ms; the index shrank 336MB -> 44MB. The old full index
-- (idx_opps_search) is then redundant and dropped.
-- NOTE: prod was migrated live via CREATE/DROP INDEX CONCURRENTLY (Neon, online).
-- CONCURRENTLY is omitted here so this file works inside a transactional runner.
CREATE INDEX IF NOT EXISTS idx_opps_search_active
  ON opportunities USING gin (search_vector)
  WHERE is_latest = true AND active = true;
DROP INDEX IF EXISTS idx_opps_search;
