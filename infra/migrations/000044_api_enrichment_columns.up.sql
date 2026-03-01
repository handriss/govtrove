ALTER TABLE opportunities ADD COLUMN middle_tier TEXT;
ALTER TABLE opportunities ADD COLUMN awardee_street_address TEXT;
ALTER TABLE opportunities ADD COLUMN awardee_city_code TEXT;
ALTER TABLE opportunities ADD COLUMN awardee_state_code TEXT;
ALTER TABLE opportunities ADD COLUMN awardee_country_code TEXT;

ALTER TABLE pipeline.snap_data_quality ADD COLUMN IF NOT EXISTS resolved BOOLEAN DEFAULT false;
