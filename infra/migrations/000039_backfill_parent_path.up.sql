-- Backfill full_parent_path_name for CSV-only records that lack API enrichment
UPDATE opportunities
SET full_parent_path_name =
  CASE
    WHEN office IS NOT NULL AND office <> '' THEN department || '.' || sub_tier || '.' || office
    WHEN sub_tier IS NOT NULL AND sub_tier <> '' THEN department || '.' || sub_tier
    ELSE department
  END
WHERE (full_parent_path_name IS NULL OR full_parent_path_name = '')
  AND department IS NOT NULL AND department <> ''
  AND is_latest = true;

-- Index for path prefix matching (used by agency filter)
CREATE INDEX IF NOT EXISTS idx_opps_parent_path
  ON opportunities(full_parent_path_name text_pattern_ops)
  WHERE full_parent_path_name IS NOT NULL AND full_parent_path_name <> '';
