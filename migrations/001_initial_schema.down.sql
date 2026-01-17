DROP TRIGGER IF EXISTS trigger_update_opportunities_updated_at ON opportunities;
DROP TRIGGER IF EXISTS trigger_update_opportunity_search_vector ON opportunities;

DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS update_opportunity_search_vector();

-- Reverse order due to FK constraints
DROP TABLE IF EXISTS ingestion_runs;
DROP TABLE IF EXISTS opportunity_resources;
DROP TABLE IF EXISTS opportunity_contacts;
DROP TABLE IF EXISTS opportunities;
