-- set_aside_code normalisation: the API ingest path let a JSON array leak into this
-- scalar column, so active rows held the literal string '["SBA"]' and a few held
-- '[""]'. Filtering set_aside=SBA silently missed all of them, under-reporting small
-- business work to the users who most depend on it.
--
-- The ingest-side guard is normalizeSetAsideCode() in
-- pipeline/internal/reconcile/from_api.go; this backfills rows written before it.
-- Unwrap single-element arrays; '[""]' collapses to empty.
UPDATE opportunities
SET set_aside_code = btrim(btrim(set_aside_code, '[]'), '"')
WHERE set_aside_code LIKE '[%]';
