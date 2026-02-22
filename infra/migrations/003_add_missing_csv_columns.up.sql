-- Add missing columns from SAM.gov CSV

-- Description text
ALTER TABLE opportunities ADD COLUMN description TEXT;

-- Agency/organization codes
ALTER TABLE opportunities ADD COLUMN department VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN sub_tier VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN office VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN cgac VARCHAR(8);
ALTER TABLE opportunities ADD COLUMN fpds_code VARCHAR(8);
ALTER TABLE opportunities ADD COLUMN aac_code VARCHAR(8);

-- Office country
ALTER TABLE opportunities ADD COLUMN office_country_code VARCHAR(3);

-- Primary contact
ALTER TABLE opportunities ADD COLUMN primary_contact_title VARCHAR(128);
ALTER TABLE opportunities ADD COLUMN primary_contact_fullname VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN primary_contact_email VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN primary_contact_phone VARCHAR(32);
ALTER TABLE opportunities ADD COLUMN primary_contact_fax VARCHAR(32);

-- Secondary contact
ALTER TABLE opportunities ADD COLUMN secondary_contact_title VARCHAR(128);
ALTER TABLE opportunities ADD COLUMN secondary_contact_fullname VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN secondary_contact_email VARCHAR(256);
ALTER TABLE opportunities ADD COLUMN secondary_contact_phone VARCHAR(32);
ALTER TABLE opportunities ADD COLUMN secondary_contact_fax VARCHAR(32);

-- Update search vector to include description
DROP TRIGGER IF EXISTS trigger_update_opportunity_search_vector ON opportunities;

CREATE OR REPLACE FUNCTION update_opportunity_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.full_parent_path_name, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_opportunity_search_vector
    BEFORE INSERT OR UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunity_search_vector();
