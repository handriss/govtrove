-- 1. Enable pg_trgm for typo-tolerant search suggestions
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 2. Reweighted search vector trigger: demote notice_id to D, add new fields to D
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

-- 3. GIN trigram indexes for typo-tolerant suggestion queries
CREATE INDEX idx_opps_title_trgm ON opportunities USING GIN (title gin_trgm_ops);
CREATE INDEX idx_opps_solnum_trgm ON opportunities USING GIN (solicitation_number gin_trgm_ops);

-- 4. Backfill existing rows (fires the new trigger)
UPDATE opportunities SET search_vector = search_vector;
