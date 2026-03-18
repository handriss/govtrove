CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE code_embeddings (
    code_type VARCHAR(5) NOT NULL,
    code VARCHAR(8) NOT NULL,
    title TEXT NOT NULL,
    level SMALLINT NOT NULL,
    embedding vector(384) NOT NULL,
    PRIMARY KEY (code_type, code)
);

CREATE MATERIALIZED VIEW code_correlations AS
SELECT
    naics_code,
    classification_code AS psc_code,
    COUNT(*) AS co_occurrence_count
FROM opportunities
WHERE naics_code IS NOT NULL AND naics_code != ''
  AND classification_code IS NOT NULL AND classification_code != ''
GROUP BY naics_code, classification_code
HAVING COUNT(*) >= 2;

CREATE UNIQUE INDEX idx_correlations_naics_psc ON code_correlations(naics_code, psc_code);
CREATE INDEX idx_correlations_naics ON code_correlations(naics_code);
CREATE INDEX idx_correlations_psc ON code_correlations(psc_code);
