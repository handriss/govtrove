ALTER TABLE opportunities ADD COLUMN data_source VARCHAR(16) DEFAULT 'csv';

CREATE INDEX idx_opportunities_data_source ON opportunities(data_source);
