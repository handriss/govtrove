-- Restore original search vector trigger (notice_id at B, set_aside_description at C)
CREATE OR REPLACE FUNCTION update_opportunities_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.department, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.sub_tier, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.office, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.awardee_name, NEW.awardee, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop trigram indexes
DROP INDEX IF EXISTS idx_opps_title_trgm;
DROP INDEX IF EXISTS idx_opps_solnum_trgm;

-- Re-backfill with original weights
UPDATE opportunities SET search_vector = search_vector;
