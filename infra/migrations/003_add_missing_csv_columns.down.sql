-- Revert search vector trigger
DROP TRIGGER IF EXISTS trigger_update_opportunity_search_vector ON opportunities;

CREATE OR REPLACE FUNCTION update_opportunity_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.full_parent_path_name, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_opportunity_search_vector
    BEFORE INSERT OR UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunity_search_vector();

-- Remove columns
ALTER TABLE opportunities DROP COLUMN IF EXISTS secondary_contact_fax;
ALTER TABLE opportunities DROP COLUMN IF EXISTS secondary_contact_phone;
ALTER TABLE opportunities DROP COLUMN IF EXISTS secondary_contact_email;
ALTER TABLE opportunities DROP COLUMN IF EXISTS secondary_contact_fullname;
ALTER TABLE opportunities DROP COLUMN IF EXISTS secondary_contact_title;

ALTER TABLE opportunities DROP COLUMN IF EXISTS primary_contact_fax;
ALTER TABLE opportunities DROP COLUMN IF EXISTS primary_contact_phone;
ALTER TABLE opportunities DROP COLUMN IF EXISTS primary_contact_email;
ALTER TABLE opportunities DROP COLUMN IF EXISTS primary_contact_fullname;
ALTER TABLE opportunities DROP COLUMN IF EXISTS primary_contact_title;

ALTER TABLE opportunities DROP COLUMN IF EXISTS office_country_code;

ALTER TABLE opportunities DROP COLUMN IF EXISTS aac_code;
ALTER TABLE opportunities DROP COLUMN IF EXISTS fpds_code;
ALTER TABLE opportunities DROP COLUMN IF EXISTS cgac;
ALTER TABLE opportunities DROP COLUMN IF EXISTS office;
ALTER TABLE opportunities DROP COLUMN IF EXISTS sub_tier;
ALTER TABLE opportunities DROP COLUMN IF EXISTS department;

ALTER TABLE opportunities DROP COLUMN IF EXISTS description;
