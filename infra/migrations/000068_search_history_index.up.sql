CREATE INDEX IF NOT EXISTS idx_opportunities_created_at_active
ON opportunities (created_at)
WHERE active = true AND is_latest = true;
