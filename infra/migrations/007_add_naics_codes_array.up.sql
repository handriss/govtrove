ALTER TABLE opportunities ADD COLUMN naics_codes TEXT[];

CREATE INDEX idx_opportunities_naics_codes ON opportunities USING GIN(naics_codes);
