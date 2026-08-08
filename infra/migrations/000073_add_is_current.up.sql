-- is_current: one row per (solicitation_number, type) among active+latest rows.
--
-- is_latest is scoped per notice_id, but SAM.gov issues a NEW notice_id for every
-- amendment/repost of the same solicitation. Each of those stays active and latest,
-- so a single solicitation accumulates many rows: 140R2026Q0014 had 6, and
-- catalog-wide 10,330 solicitations accounted for 40,261 of 67,130 active rows.
-- Users saw it as result lists repeating the same notice (a set-aside search
-- returned "8 results" that were one solicitation six times).
--
-- Keyed on (solicitation_number, type), NOT solicitation alone: 3,196 groups mix
-- types, e.g. a Sources Sought whose Solicitation posted later. Those are genuinely
-- distinct stages and collapsing them would hide the pre-solicitation window, which
-- is exactly what consultants watch for.
--
-- Defaults to true so behaviour is unchanged until reconcile populates it.
ALTER TABLE opportunities
  ADD COLUMN IF NOT EXISTS is_current boolean NOT NULL DEFAULT true;

-- Supports the reconcile pass that recomputes the flag.
CREATE INDEX IF NOT EXISTS idx_opps_soltype_current
  ON opportunities (solicitation_number, type, posted_date DESC)
  WHERE is_latest = true AND active = true;

-- set_aside_code normalisation: the API ingest path let a JSON array leak into this
-- scalar column, so 178 active rows held the literal string '["SBA"]' and 12 held
-- '[""]'. Filtering set_aside=SBA silently missed all 178, under-reporting small
-- business work. Unwrap single-element arrays; treat '[""]' as empty.
UPDATE opportunities
SET set_aside_code = btrim(btrim(set_aside_code, '[]'), '"')
WHERE set_aside_code LIKE '[%]';
