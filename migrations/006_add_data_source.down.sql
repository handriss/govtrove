DROP INDEX IF EXISTS idx_opportunities_data_source;

ALTER TABLE opportunities DROP COLUMN IF EXISTS data_source;
