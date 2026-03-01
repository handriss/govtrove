package e2e_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/handriss/govtrove/pipeline/internal/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pipeline E2E Suite")
}

var (
	db       *database.DB
	pgCtr    *postgres.PostgresContainer
	dbURL    string
	logger   *slog.Logger
	suiteCtx context.Context
)

var _ = BeforeSuite(func() {
	suiteCtx = context.Background()
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	var err error
	pgCtr, err = postgres.Run(suiteCtx,
		"postgres:16-alpine",
		postgres.WithDatabase("govtrove_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	Expect(err).NotTo(HaveOccurred())

	dbURL, err = pgCtr.ConnectionString(suiteCtx, "sslmode=disable")
	Expect(err).NotTo(HaveOccurred())

	db, err = database.New(suiteCtx, dbURL)
	Expect(err).NotTo(HaveOccurred())

	_, err = db.Pool().Exec(suiteCtx, schemaSQL)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	if db != nil {
		db.Close()
	}
	if pgCtr != nil {
		_ = pgCtr.Terminate(suiteCtx)
	}
})

// schemaSQL creates the tables the pipeline uses. Derived from the production
// migrations but written as a single idempotent DDL block so the e2e suite
// doesn't depend on the full migration chain (which has rename/drop steps
// that only work against the production DB's history).
const schemaSQL = `
CREATE SCHEMA IF NOT EXISTS pipeline;

-- Pipeline runs (must be before ingestion_runs due to FK)
CREATE TABLE pipeline.pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms INT,
    stats JSONB,
    error_message TEXT,
    metadata JSONB
);

-- Ingestion runs
CREATE TABLE pipeline.ingestion_runs (
    run_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type        TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'running',
    records_fetched   INT DEFAULT 0,
    records_inserted  INT DEFAULT 0,
    records_updated   INT DEFAULT 0,
    records_failed    INT DEFAULT 0,
    records_skipped   INT DEFAULT 0,
    total_db_count    INT DEFAULT 0,
    error_message   TEXT,
    duration_ms     INT,
    pipeline_run_id UUID REFERENCES pipeline.pipeline_runs(id)
);
CREATE INDEX idx_ingestion_runs_status ON pipeline.ingestion_runs(status);
CREATE INDEX idx_ingestion_runs_job_type ON pipeline.ingestion_runs(job_type, started_at DESC);

-- Snapshot CSV
CREATE TABLE pipeline.snap_csv (
    id              BIGSERIAL PRIMARY KEY,
    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    title           TEXT,
    type            TEXT,
    base_type       TEXT,
    posted_date     TIMESTAMPTZ,
    response_deadline TIMESTAMPTZ,
    archive_date    TEXT,
    archive_type    TEXT,
    set_aside_code  TEXT,
    naics_code      TEXT,
    classification_code TEXT,
    active          BOOLEAN,
    department      TEXT,
    sub_tier        TEXT,
    office          TEXT,
    cgac            TEXT,
    fpds_code       TEXT,
    aac_code        TEXT,
    award_number    TEXT,
    award_date      TEXT,
    award_amount    NUMERIC(15,2),
    raw_data        JSONB NOT NULL,
    content_hash    TEXT NOT NULL,
    run_id          UUID NOT NULL REFERENCES pipeline.ingestion_runs(run_id),
    snapshot_date   TIMESTAMPTZ NOT NULL,
    download_id     BIGINT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(notice_id, run_id)
);
CREATE INDEX idx_snap_csv_notice_id ON pipeline.snap_csv(notice_id);
CREATE INDEX idx_snap_csv_run_id ON pipeline.snap_csv(run_id);
CREATE INDEX idx_snap_csv_hash ON pipeline.snap_csv(notice_id, run_id, content_hash);

-- Disappearances
CREATE TABLE pipeline.snap_disappearances (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL REFERENCES pipeline.ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    solicitation_number TEXT,
    last_seen_date  TIMESTAMPTZ NOT NULL,
    disappeared_date TIMESTAMPTZ NOT NULL,
    last_type       TEXT,
    last_archive_type TEXT,
    last_archive_date TEXT,
    resolution      TEXT,
    resolution_date TIMESTAMPTZ,
    resolution_source TEXT,
    reappeared_date TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_snap_disapp_run_id ON pipeline.snap_disappearances(run_id);
CREATE INDEX idx_snap_disapp_notice ON pipeline.snap_disappearances(notice_id);

-- Data quality
CREATE TABLE pipeline.snap_data_quality (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL REFERENCES pipeline.ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    snapshot_date   TIMESTAMPTZ NOT NULL,
    source          TEXT NOT NULL,
    issue_type      TEXT NOT NULL,
    field_name      TEXT,
    field_value     TEXT,
    description     TEXT,
    resolved        BOOLEAN DEFAULT false,
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_snap_dq_run_id ON pipeline.snap_data_quality(run_id);

-- Reconcile data quality
CREATE TABLE pipeline.snap_reconcile_dq (
    id              BIGSERIAL PRIMARY KEY,
    csv_run_id      UUID REFERENCES pipeline.ingestion_runs(run_id),
    api_run_id      UUID REFERENCES pipeline.ingestion_runs(run_id),
    notice_id       TEXT NOT NULL,
    snapshot_date   TIMESTAMPTZ NOT NULL,
    issue_type      TEXT NOT NULL,
    field_name      TEXT NOT NULL,
    csv_value       TEXT,
    api_value       TEXT,
    resolved        BOOLEAN DEFAULT false,
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Opportunities
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

-- Search vector trigger
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

-- Updated_at trigger
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
`
