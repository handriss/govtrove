-- Revert varchar expansions
ALTER TABLE opportunities ALTER COLUMN pop_zip TYPE VARCHAR(10);
ALTER TABLE opportunities ALTER COLUMN office_zip TYPE VARCHAR(10);
ALTER TABLE opportunities ALTER COLUMN awardee_zip TYPE VARCHAR(10);
ALTER TABLE opportunities ALTER COLUMN pop_state_code TYPE VARCHAR(10);
ALTER TABLE opportunities ALTER COLUMN pop_country_code TYPE VARCHAR(3);
ALTER TABLE opportunities ALTER COLUMN office_country_code TYPE VARCHAR(3);
ALTER TABLE opportunities ALTER COLUMN pop_city_code TYPE VARCHAR(32);
