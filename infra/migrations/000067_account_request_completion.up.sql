ALTER TABLE account_requests
  ADD COLUMN completed_at TIMESTAMPTZ,
  ADD COLUMN completed_by VARCHAR(255),
  ADD COLUMN export_s3_key VARCHAR(500);
