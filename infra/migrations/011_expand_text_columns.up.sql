-- Expand varchar(256) columns to TEXT to handle long values from SAM.gov CSV
ALTER TABLE opportunities ALTER COLUMN set_aside_description TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN awardee_name TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN department TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN sub_tier TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN office TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN primary_contact_fullname TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN primary_contact_email TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN secondary_contact_fullname TYPE TEXT;
ALTER TABLE opportunities ALTER COLUMN secondary_contact_email TYPE TEXT;
