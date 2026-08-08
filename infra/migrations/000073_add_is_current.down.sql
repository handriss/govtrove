-- The set_aside_code normalisation is not reversed: the '["SBA"]' form was a bug,
-- and restoring it would re-break set-aside filtering.
DROP INDEX IF EXISTS idx_opps_soltype_current;
ALTER TABLE opportunities DROP COLUMN IF EXISTS is_current;
