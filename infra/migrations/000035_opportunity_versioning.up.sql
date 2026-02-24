ALTER TABLE opportunities ADD COLUMN version INT NOT NULL DEFAULT 1;
ALTER TABLE opportunities ADD COLUMN is_latest BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE opportunities ADD COLUMN content_hash TEXT;

ALTER TABLE opportunities DROP CONSTRAINT opportunities_notice_id_key;
ALTER TABLE opportunities ADD CONSTRAINT opportunities_notice_id_version_key UNIQUE (notice_id, version);

CREATE INDEX idx_opps_is_latest ON opportunities (is_latest) WHERE is_latest = true;
