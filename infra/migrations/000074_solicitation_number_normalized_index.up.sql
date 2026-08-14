-- Known-item lookup: a pasted notice number is matched against the solicitation
-- number with punctuation stripped from both sides, because ~13% of stored
-- numbers carry dashes and users paste whichever form they were given.
--
-- Without this index that predicate is a per-row regexp over the whole active
-- set — 2.6s on /opportunities and 6.4s on /opportunities/facets, which runs the
-- predicate once per facet dimension. The expression must match the one in
-- buildFilterConditions exactly or the planner won't use it.
--
-- Partial to active+latest: every search path already pins both, so this indexes
-- ~72k rows rather than ~540k.
-- NOTE: applied to prod with CREATE INDEX CONCURRENTLY (omitted here so the file
-- works inside a transactional migration runner).
CREATE INDEX IF NOT EXISTS idx_opps_solnum_normalized
  ON opportunities (regexp_replace(upper(COALESCE(solicitation_number, '')), '[^A-Z0-9]', '', 'g'))
  WHERE is_latest = true AND active = true;
