ALTER TABLE opportunities ADD COLUMN ingestion_run_id INTEGER REFERENCES ingestion_runs(id);

CREATE INDEX idx_opportunities_ingestion_run ON opportunities(ingestion_run_id);
