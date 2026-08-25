-- Renumbered from 022 (legacy 3-digit) which collided with 000022 and made
-- golang-migrate refuse to parse the directory at all. Applied by hand in
-- production before the renumber, so every statement is idempotent.
CREATE TABLE IF NOT EXISTS geo_synonyms (
    id SERIAL PRIMARY KEY,
    term TEXT NOT NULL,
    match_type TEXT NOT NULL,
    states TEXT[],
    cities TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_geo_synonyms_term ON geo_synonyms (lower(term));
