-- Expand state code column (some records have longer values than 2 chars)
ALTER TABLE opportunities ALTER COLUMN pop_state_code TYPE VARCHAR(10);

-- Expand phone/fax columns (can have extensions that exceed 32 chars)
ALTER TABLE opportunities ALTER COLUMN primary_contact_phone TYPE VARCHAR(64);
ALTER TABLE opportunities ALTER COLUMN primary_contact_fax TYPE VARCHAR(64);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_phone TYPE VARCHAR(64);
ALTER TABLE opportunities ALTER COLUMN secondary_contact_fax TYPE VARCHAR(64);
