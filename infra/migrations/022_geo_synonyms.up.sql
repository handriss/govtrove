CREATE TABLE geo_synonyms (
    id SERIAL PRIMARY KEY,
    term TEXT NOT NULL,
    match_type TEXT NOT NULL,
    states TEXT[],
    cities TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_geo_synonyms_term ON geo_synonyms (lower(term));
