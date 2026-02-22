-- Expand remaining small varchar columns that can exceed their limits
-- Zip codes can be international or have extensions
ALTER TABLE opportunities ALTER COLUMN pop_zip TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN office_zip TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN awardee_zip TYPE VARCHAR(32);

-- State codes can sometimes have longer values
ALTER TABLE opportunities ALTER COLUMN pop_state_code TYPE VARCHAR(32);

-- Country codes can be ISO-3 or full names in some cases
ALTER TABLE opportunities ALTER COLUMN pop_country_code TYPE VARCHAR(32);
ALTER TABLE opportunities ALTER COLUMN office_country_code TYPE VARCHAR(32);

-- City codes can be longer
ALTER TABLE opportunities ALTER COLUMN pop_city_code TYPE VARCHAR(64);
