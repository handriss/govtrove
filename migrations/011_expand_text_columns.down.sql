-- Note: This may fail if existing data exceeds 256 chars
ALTER TABLE opportunities ALTER COLUMN set_aside_description TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN awardee_name TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN department TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN sub_tier TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN office TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN primary_contact_fullname TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN primary_contact_email TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_fullname TYPE VARCHAR(256);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_email TYPE VARCHAR(256);
