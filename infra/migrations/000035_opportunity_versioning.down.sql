DROP INDEX IF EXISTS idx_opps_is_latest;

ALTER TABLE opportunities DROP CONSTRAINT IF EXISTS opportunities_notice_id_version_key;
ALTER TABLE opportunities ADD CONSTRAINT opportunities_notice_id_key UNIQUE (notice_id);

ALTER TABLE opportunities DROP COLUMN IF EXISTS content_hash;
ALTER TABLE opportunities DROP COLUMN IF EXISTS is_latest;
ALTER TABLE opportunities DROP COLUMN IF EXISTS version;
