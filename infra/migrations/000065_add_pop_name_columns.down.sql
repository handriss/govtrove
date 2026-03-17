-- Restore old search vector trigger (without pop_state_name/pop_country_name)
CREATE OR REPLACE FUNCTION update_opportunities_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.department, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.sub_tier, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.office, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.awardee_name, NEW.awardee, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.classification_code, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.naics_code, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.pop_city, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.award_number, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.primary_contact_fullname, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Rebuild search vectors
UPDATE opportunities SET search_vector = search_vector;

-- Drop columns
ALTER TABLE opportunities DROP COLUMN pop_state_name;
ALTER TABLE opportunities DROP COLUMN pop_country_name;
