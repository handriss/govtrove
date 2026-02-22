CREATE TABLE opportunities (
    id SERIAL PRIMARY KEY,
    notice_id VARCHAR(64) UNIQUE NOT NULL,
    solicitation_number VARCHAR(128),
    title TEXT NOT NULL,
    description_url TEXT,
    ui_link TEXT,
    additional_info_link TEXT,
    type VARCHAR(64),
    base_type VARCHAR(64),
    organization_type VARCHAR(64),
    full_parent_path_name TEXT,
    full_parent_path_code TEXT,
    posted_date TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date TIMESTAMPTZ,
    archive_type VARCHAR(32),
    set_aside_code VARCHAR(32),
    set_aside_description VARCHAR(256),
    naics_code VARCHAR(6),
    classification_code VARCHAR(8),
    pop_street_address TEXT,
    pop_street_address2 TEXT,
    pop_city_code VARCHAR(32),
    pop_city_name VARCHAR(128),
    pop_state_code VARCHAR(2),
    pop_state_name VARCHAR(64),
    pop_country_code VARCHAR(3),
    pop_country_name VARCHAR(64),
    pop_zip VARCHAR(10),
    office_city VARCHAR(128),
    office_state VARCHAR(64),
    office_zip VARCHAR(10),
    award_number VARCHAR(64),
    award_amount NUMERIC(15,2),
    award_date DATE,
    awardee_name VARCHAR(256),
    awardee_uei VARCHAR(12),
    awardee_street_address TEXT,
    awardee_street_address2 TEXT,
    awardee_city VARCHAR(128),
    awardee_state VARCHAR(64),
    awardee_country VARCHAR(64),
    awardee_zip VARCHAR(10),
    active BOOLEAN DEFAULT true,
    raw_json JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    search_vector TSVECTOR
);

CREATE TABLE opportunity_contacts (
    id SERIAL PRIMARY KEY,
    opportunity_id INTEGER NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    contact_type VARCHAR(64),
    title VARCHAR(128),
    full_name VARCHAR(256),
    email VARCHAR(256),
    phone VARCHAR(32),
    fax VARCHAR(32),
    additional_info TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE opportunity_resources (
    id SERIAL PRIMARY KEY,
    opportunity_id INTEGER NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ingestion_runs (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status VARCHAR(32) DEFAULT 'running',
    records_fetched INTEGER DEFAULT 0,
    records_inserted INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER
);

CREATE INDEX idx_opportunities_notice_id ON opportunities(notice_id);
CREATE INDEX idx_opportunities_solicitation_number ON opportunities(solicitation_number);
CREATE INDEX idx_opportunities_posted_date ON opportunities(posted_date DESC);
CREATE INDEX idx_opportunities_response_deadline ON opportunities(response_deadline);
CREATE INDEX idx_opportunities_type ON opportunities(type);
CREATE INDEX idx_opportunities_base_type ON opportunities(base_type);
CREATE INDEX idx_opportunities_set_aside ON opportunities(set_aside_code);
CREATE INDEX idx_opportunities_naics ON opportunities(naics_code);
CREATE INDEX idx_opportunities_classification ON opportunities(classification_code);
CREATE INDEX idx_opportunities_active ON opportunities(active);
CREATE INDEX idx_opportunities_search ON opportunities USING GIN(search_vector);
CREATE INDEX idx_opportunities_raw_json ON opportunities USING GIN(raw_json);

CREATE INDEX idx_opportunity_contacts_opportunity ON opportunity_contacts(opportunity_id);
CREATE INDEX idx_opportunity_resources_opportunity ON opportunity_resources(opportunity_id);
CREATE INDEX idx_ingestion_runs_status ON ingestion_runs(status);
CREATE INDEX idx_ingestion_runs_started_at ON ingestion_runs(started_at DESC);

CREATE OR REPLACE FUNCTION update_opportunity_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.full_parent_path_name, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_opportunity_search_vector
    BEFORE INSERT OR UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunity_search_vector();

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_opportunities_updated_at
    BEFORE UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
