package searchtest_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

func searchGet(t *testing.T, baseURL string, params url.Values) models.SearchResult {
	t.Helper()
	u := baseURL + "/api/opportunities"
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", u, resp.StatusCode)
	}
	var result models.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func titles(result models.SearchResult) []string {
	out := make([]string, len(result.Opportunities))
	for i, o := range result.Opportunities {
		out[i] = o.Title
	}
	return out
}

func hasTitle(result models.SearchResult, title string) bool {
	for _, o := range result.Opportunities {
		if o.Title == title {
			return true
		}
	}
	return false
}

func ptr(s string) *string { return &s }

func timePtr(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("bad time: " + s)
	}
	return &t
}

const schemaSQL = `
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE opportunities (
    id              BIGSERIAL PRIMARY KEY,
    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    title           TEXT,
    description     TEXT,
    type            TEXT,
    base_type       TEXT,
    organization_type TEXT,
    posted_date     TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date    DATE,
    archive_type    TEXT,
    active          BOOLEAN DEFAULT true,
    set_aside_code  TEXT,
    set_aside_description TEXT,
    naics_code      TEXT,
    classification_code TEXT,
    department      TEXT,
    sub_tier        TEXT,
    office          TEXT,
    cgac            TEXT,
    fpds_code       TEXT,
    aac_code        TEXT,
    middle_tier     TEXT,
    full_parent_path_name TEXT,
    full_parent_path_code TEXT,
    pop_street_address TEXT,
    pop_city        TEXT,
    pop_state       TEXT,
    pop_zip         TEXT,
    pop_country     TEXT,
    pop_city_code   TEXT,
    pop_state_code  TEXT,
    pop_country_code TEXT,
    office_city     TEXT,
    office_state    TEXT,
    office_zip      TEXT,
    office_country  TEXT,
    award_number    TEXT,
    award_date      DATE,
    award_amount    NUMERIC(15,2),
    awardee         TEXT,
    awardee_name    TEXT,
    awardee_uei     TEXT,
    awardee_cage_code TEXT,
    awardee_street_address TEXT,
    awardee_city    TEXT,
    awardee_city_code TEXT,
    awardee_state   TEXT,
    awardee_state_code TEXT,
    awardee_zip     TEXT,
    awardee_country TEXT,
    awardee_country_code TEXT,
    primary_contact_title TEXT,
    primary_contact_fullname TEXT,
    primary_contact_email TEXT,
    primary_contact_phone TEXT,
    primary_contact_fax TEXT,
    secondary_contact_title TEXT,
    secondary_contact_fullname TEXT,
    secondary_contact_email TEXT,
    secondary_contact_phone TEXT,
    secondary_contact_fax TEXT,
    ui_link         TEXT,
    additional_info_link TEXT,
    description_url TEXT,
    resource_links  JSONB,
    version         INT NOT NULL DEFAULT 1,
    is_latest       BOOLEAN NOT NULL DEFAULT true,
    content_hash    TEXT,
    data_sources    TEXT DEFAULT 'csv',
    last_csv_run_id UUID,
    last_api_run_id UUID,
    first_seen_at   TIMESTAMPTZ DEFAULT NOW(),
    last_seen_csv   TIMESTAMPTZ,
    last_seen_api   TIMESTAMPTZ,
    search_vector   TSVECTOR,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_opps_notice_version ON opportunities(notice_id, version);
CREATE INDEX idx_opps_notice_id ON opportunities(notice_id);
CREATE INDEX idx_opps_is_latest ON opportunities(is_latest) WHERE is_latest = true;
CREATE INDEX idx_opps_search ON opportunities USING GIN(search_vector);
CREATE INDEX idx_opps_title_trgm ON opportunities USING GIN (title gin_trgm_ops);
CREATE INDEX idx_opps_solnum_trgm ON opportunities USING GIN (solicitation_number gin_trgm_ops);

CREATE OR REPLACE FUNCTION update_opportunities_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.solicitation_number, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.department, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.sub_tier, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.office, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.notice_id, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.set_aside_description, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.awardee_name, NEW.awardee, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.classification_code, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.naics_code, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.pop_city, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.award_number, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.primary_contact_fullname, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_opps_search_vector
    BEFORE INSERT OR UPDATE ON opportunities
    FOR EACH ROW
    EXECUTE FUNCTION update_opportunities_search_vector();

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

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    workos_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(320) NOT NULL,
    first_name VARCHAR(200),
    last_name VARCHAR(200),
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    stripe_customer_id TEXT UNIQUE,
    subscription_id TEXT,
    subscription_status TEXT,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    current_period_end TIMESTAMPTZ,
    free_forever BOOLEAN NOT NULL DEFAULT FALSE,
    gift_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE search_events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(32) NOT NULL,
    user_id INTEGER REFERENCES users(id),
    query TEXT,
    filters JSONB,
    sort_by VARCHAR(32),
    page INTEGER,
    total_results INTEGER,
    opportunity_id INTEGER,
    duration_ms INTEGER,
    user_agent TEXT,
    referer TEXT,
    utm_campaign TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);
`

const seedSQL = `
INSERT INTO opportunities (notice_id, title, description, solicitation_number, type, posted_date, response_deadline, active, is_latest, set_aside_code, set_aside_description, naics_code, classification_code, department, pop_state)
VALUES
    ('TEST-001', 'Sterilizer Equipment Maintenance', 'Maintenance and repair of sterilizer equipment for medical facilities', 'SOL-2026-001', 'Solicitation', '2026-03-01', '2026-04-01', true, true, 'SBA', 'Total Small Business Set-Aside', '339113', 'J065', 'DEPT OF DEFENSE', 'VA'),
    ('TEST-002', 'Cybersecurity Assessment Services', 'Comprehensive cybersecurity assessment and penetration testing services', 'SOL-2026-002', 'Sources Sought', '2026-03-05', '2026-04-15', true, true, 'SDVOSBC', 'Service-Disabled Veteran-Owned Small Business', '541512', 'D399', 'DEPT OF HOMELAND SECURITY', 'DC'),
    ('TEST-003', 'Autoclave Repair and Calibration', 'Repair and calibration of autoclave sterilization equipment', 'SOL-2026-003', 'Presolicitation', '2026-02-15', '2026-03-15', true, true, '8AN', '8(a) Sole Source', '339113', 'J065', 'DEPT OF VETERANS AFFAIRS', 'CA'),
    ('TEST-004', 'Cloud Infrastructure Migration', 'Migration of legacy systems to cloud infrastructure', 'SOL-2026-004', 'Combined Synopsis/Solicitation', '2026-03-10', '2026-05-01', true, true, NULL, NULL, '541519', 'D304', 'GENERAL SERVICES ADMINISTRATION', 'NY'),
    ('TEST-005', 'Janitorial Services for Federal Building', 'Janitorial and custodial services for a federal office building', 'SOL-2026-005', 'Solicitation', '2026-01-15', '2026-02-28', true, true, 'SBA', 'Total Small Business Set-Aside', '561720', 'S216', 'GENERAL SERVICES ADMINISTRATION', 'TX'),
    ('TEST-006', 'Sterilizer Supply Chain Analysis', 'Analysis of sterilizer supply chain logistics and procurement', NULL, 'Award Notice', '2026-03-08', NULL, true, true, NULL, NULL, '339113', NULL, 'DEPT OF DEFENSE', 'MD'),
    ('TEST-007', 'Network Security Monitoring Tools', 'Procurement of network security monitoring and intrusion detection tools', 'SOL-2026-007', 'Special Notice', '2026-03-12', '2026-06-01', true, true, 'WOSB', 'Women-Owned Small Business', '541512', 'D399', 'DEPT OF DEFENSE', 'VA'),
    ('TEST-008', 'Medical Laboratory Equipment', 'Supply of medical laboratory testing equipment and supplies', 'SOL-2026-008', 'Solicitation', '2026-03-03', '2026-04-10', true, true, 'SBA', 'Total Small Business Set-Aside', '339112', 'A075', 'DEPT OF DEFENSE', NULL),
    ('TEST-009', 'Inactive Sterilizer Opportunity', 'This opportunity is no longer active', 'SOL-2026-009', 'Solicitation', '2026-03-01', '2026-04-01', false, true, 'SBA', 'Total Small Business Set-Aside', '541512', NULL, 'DEPT OF DEFENSE', 'VA'),
    ('TEST-010', 'Old Version Opportunity', 'This is an older version of an opportunity', 'SOL-2026-010', 'Solicitation', '2026-03-01', '2026-04-01', true, false, 'SBA', 'Total Small Business Set-Aside', '541512', NULL, 'DEPT OF DEFENSE', 'VA');
`
