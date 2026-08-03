DROP TRIGGER IF EXISTS trigger_opps_search_vector ON opportunities;

CREATE TRIGGER trigger_opps_search_vector
BEFORE INSERT OR UPDATE
ON opportunities
FOR EACH ROW
EXECUTE FUNCTION update_opportunities_search_vector();
