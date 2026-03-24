ALTER TABLE account_requests
  DROP COLUMN IF EXISTS completed_at,
  DROP COLUMN IF EXISTS completed_by,
  DROP COLUMN IF EXISTS export_s3_key;
