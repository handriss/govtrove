CREATE TABLE usaspending_award_summaries (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    naics_code VARCHAR(6) NOT NULL,
    sub_tier_agency_code VARCHAR(4) NOT NULL,
    sub_tier_agency_name TEXT NOT NULL,
    total_award_count INTEGER NOT NULL DEFAULT 0,
    set_aside_award_count INTEGER NOT NULL DEFAULT 0,
    unique_awardee_count INTEGER NOT NULL DEFAULT 0,
    total_obligated_amount NUMERIC NOT NULL DEFAULT 0,
    fiscal_year INTEGER NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (naics_code, sub_tier_agency_code, fiscal_year)
);

CREATE INDEX idx_award_summaries_lookup
    ON usaspending_award_summaries (naics_code, sub_tier_agency_code);
CREATE INDEX idx_award_summaries_fetched
    ON usaspending_award_summaries (fetched_at);

CREATE TABLE usaspending_award_recipients (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    summary_id INTEGER NOT NULL REFERENCES usaspending_award_summaries(id) ON DELETE CASCADE,
    recipient_name TEXT NOT NULL,
    recipient_uei VARCHAR(12),
    total_obligated_amount NUMERIC NOT NULL DEFAULT 0,
    award_count INTEGER NOT NULL DEFAULT 0,
    business_type TEXT,
    rank SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_award_recipients_summary
    ON usaspending_award_recipients (summary_id);
