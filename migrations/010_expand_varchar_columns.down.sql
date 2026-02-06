-- Note: This may fail if existing data exceeds the original limits
ALTER TABLE opportunities ALTER COLUMN pop_state_code TYPE VARCHAR(2);
ALTER TABLE opportunities ALTER COLUMN primary_contact_phone TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN primary_contact_fax TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_phone TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_fax TYPE VARCHAR(32);
