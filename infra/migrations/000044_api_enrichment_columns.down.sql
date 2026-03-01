ALTER TABLE opportunities DROP COLUMN IF EXISTS middle_tier;
ALTER TABLE opportunities DROP COLUMN IF EXISTS awardee_street_address;
ALTER TABLE opportunities DROP COLUMN IF EXISTS awardee_city_code;
ALTER TABLE opportunities DROP COLUMN IF EXISTS awardee_state_code;
ALTER TABLE opportunities DROP COLUMN IF EXISTS awardee_country_code;

ALTER TABLE pipeline.snap_data_quality DROP COLUMN IF EXISTS resolved;
