-- ============================================================
-- LAYER 3: CANONICAL OPPORTUNITIES TABLE
-- ============================================================
-- One row per notice_id. The query-facing table for frontend/API.
-- Populated by reconciling data from snap_csv (and later snap_api).
-- Can always be rebuilt from snapshot tables.
--
-- Design:
--   - Superset of CSV and API fields
--   - API-only columns stay NULL until API pipeline is built
--   - Contacts denormalized (primary + secondary flat columns)
--   - resource_links as JSONB array (API-only)
--   - Full-text search via TSVECTOR with weighted fields
--   - Source tracking: which pipelines contributed, when
--
-- Column sizing lesson from v1: use TEXT wherever SAM.gov
-- data has surprised us with length. Only use VARCHAR where
-- there's a genuine upper bound (codes, zips, country codes).
-- ============================================================

CREATE TABLE opportunities (
    id              BIGSERIAL PRIMARY KEY,
    notice_id       VARCHAR(64) UNIQUE NOT NULL,

    -- === Core metadata ===
    solicitation_number VARCHAR(128),
    title           TEXT,
    description     TEXT,
    type            VARCHAR(64),
    base_type       VARCHAR(64),
    organization_type VARCHAR(64),

    -- === Dates ===
    posted_date     TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date    DATE,
    archive_type    VARCHAR(32),
    active          BOOLEAN DEFAULT true,

    -- === Classification ===
    set_aside_code  VARCHAR(32),
    set_aside_description TEXT,
    naics_code      VARCHAR(6),
    classification_code VARCHAR(8),           -- PSC code

    -- === Organization (CSV flat fields) ===
    department      TEXT,
    sub_tier        TEXT,
    office          TEXT,
    cgac            VARCHAR(8),               -- CSV-only
    fpds_code       VARCHAR(16),              -- CSV-only
    aac_code        VARCHAR(16),              -- CSV-only

    -- === Organization (API structured fields — NULL until API pipeline) ===
    full_parent_path_name TEXT,               -- "DEPT OF DEFENSE.DEPT OF THE NAVY..."
    full_parent_path_code TEXT,               -- "017.1700.N00023"

    -- === Place of Performance ===
    pop_street_address TEXT,
    pop_city        VARCHAR(128),
    pop_state       VARCHAR(32),
    pop_zip         VARCHAR(32),
    pop_country     VARCHAR(32),

    -- === Office Address ===
    office_city     VARCHAR(128),
    office_state    VARCHAR(64),
    office_zip      VARCHAR(32),
    office_country  VARCHAR(32),

    -- === Award ===
    award_number    VARCHAR(128),
    award_date      DATE,
    award_amount    NUMERIC(15,2),

    -- === Awardee (flat, from CSV) ===
    awardee         TEXT,                     -- "COMPANY NAME City ST Zip Country"

    -- === Awardee (structured, from API — NULL until API pipeline) ===
    awardee_name    TEXT,
    awardee_uei     VARCHAR(12),              -- UEI SAM
    awardee_cage_code VARCHAR(8),
    awardee_city    VARCHAR(128),
    awardee_state   VARCHAR(64),
    awardee_zip     VARCHAR(32),
    awardee_country VARCHAR(64),

    -- === Contacts (denormalized) ===
    primary_contact_title TEXT,
    primary_contact_fullname TEXT,
    primary_contact_email TEXT,
    primary_contact_phone VARCHAR(64),
    primary_contact_fax VARCHAR(64),

    secondary_contact_title TEXT,
    secondary_contact_fullname TEXT,
    secondary_contact_email TEXT,
    secondary_contact_phone VARCHAR(64),
    secondary_contact_fax VARCHAR(64),

    -- === Links ===
    ui_link         TEXT,                     -- SAM.gov UI link
    additional_info_link TEXT,
    description_url TEXT,                     -- API: URL to fetch full description
    resource_links  JSONB,                    -- API-only: array of attachment URLs

    -- === Source tracking ===
    data_sources    VARCHAR(16) DEFAULT 'csv', -- 'csv', 'api', 'csv+api'
    last_csv_run_id UUID,
    last_api_run_id UUID,
    first_seen_at   TIMESTAMPTZ DEFAULT NOW(),
    last_seen_csv   TIMESTAMPTZ,
    last_seen_api   TIMESTAMPTZ,

    -- === Full-text search ===
    search_vector   TSVECTOR,

    -- === Timestamps ===
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- === Indexes ===

-- Primary lookups
CREATE INDEX idx_opps_notice_id ON opportunities(notice_id);
CREATE INDEX idx_opps_sol_num ON opportunities(solicitation_number)
    WHERE solicitation_number IS NOT NULL;

-- Filtering
CREATE INDEX idx_opps_type ON opportunities(type);
CREATE INDEX idx_opps_base_type ON opportunities(base_type);
CREATE INDEX idx_opps_set_aside ON opportunities(set_aside_code)
    WHERE set_aside_code IS NOT NULL;
CREATE INDEX idx_opps_naics ON opportunities(naics_code)
    WHERE naics_code IS NOT NULL;
CREATE INDEX idx_opps_classification ON opportunities(classification_code)
    WHERE classification_code IS NOT NULL;
CREATE INDEX idx_opps_active ON opportunities(active);

-- Date-based queries
CREATE INDEX idx_opps_posted_date ON opportunities(posted_date DESC);
CREATE INDEX idx_opps_response_deadline ON opportunities(response_deadline);
CREATE INDEX idx_opps_archive_date ON opportunities(archive_date);

-- Full-text search
CREATE INDEX idx_opps_search ON opportunities USING GIN(search_vector);

-- Source tracking
CREATE INDEX idx_opps_data_sources ON opportunities(data_sources);

-- JSONB (for resource_links queries when API data flows)
CREATE INDEX idx_opps_resource_links ON opportunities USING GIN(resource_links)
    WHERE resource_links IS NOT NULL;

-- === Triggers ===

CREATE OR REPLACE FUNCTION update_opportunities_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_opps_updated_at
    BEFORE UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunities_updated_at();

CREATE OR REPLACE FUNCTION update_opportunities_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.department, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.sub_tier, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.office, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.awardee_name, NEW.awardee, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_opps_search_vector
    BEFORE INSERT OR UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunities_search_vector();
