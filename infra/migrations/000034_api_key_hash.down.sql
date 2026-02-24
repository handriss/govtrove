DROP TABLE IF EXISTS pipeline.api_keys;
ALTER TABLE pipeline.samgov_requests DROP COLUMN IF EXISTS api_key_hash;
