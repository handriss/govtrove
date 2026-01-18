DROP INDEX IF EXISTS idx_opportunities_naics_codes;

ALTER TABLE opportunities DROP COLUMN naics_codes;
