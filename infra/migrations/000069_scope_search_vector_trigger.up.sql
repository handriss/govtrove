-- trigger_opps_search_vector fired BEFORE INSERT OR UPDATE unconditionally,
-- recomputing 17 to_tsvector('english', ...) fields + GIN index maintenance on
-- every row touched. The daily reconcile BulkTouchUnchanged UPDATE only bumps
-- last_csv_run_id/last_seen_csv/active/data_sources on ~77K unchanged rows, yet
-- it triggered a full search-vector rebuild on each — ~12 min of wasted stemming
-- that pushed the reconcile Lambda past its 900s (hard max) timeout.
--
-- Scope the trigger to the columns the search vector is actually derived from so
-- it only fires when one of them is written. UPDATEs that touch only bookkeeping
-- columns no longer recompute the vector.
DROP TRIGGER IF EXISTS trigger_opps_search_vector ON opportunities;

CREATE TRIGGER trigger_opps_search_vector
BEFORE INSERT OR UPDATE OF
    title, solicitation_number, description, department, sub_tier, office,
    notice_id, set_aside_description, awardee_name, awardee, classification_code,
    naics_code, pop_city, pop_state_name, pop_country_name, award_number,
    primary_contact_fullname
ON opportunities
FOR EACH ROW
EXECUTE FUNCTION update_opportunities_search_vector();
