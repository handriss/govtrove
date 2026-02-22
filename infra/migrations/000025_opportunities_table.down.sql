DROP TRIGGER IF EXISTS trigger_opps_search_vector ON opportunities;
DROP TRIGGER IF EXISTS trigger_opps_updated_at ON opportunities;
DROP FUNCTION IF EXISTS update_opportunities_search_vector();
DROP FUNCTION IF EXISTS update_opportunities_updated_at();
DROP TABLE IF EXISTS opportunities;
