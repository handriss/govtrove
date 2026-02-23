-- Step 1a: Extend saved_searches with alerting columns
ALTER TABLE saved_searches
    ADD COLUMN alert_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN last_checked_at TIMESTAMPTZ,
    ADD COLUMN last_match_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN total_result_count INTEGER;

CREATE INDEX idx_saved_searches_alert_enabled
    ON saved_searches (id) WHERE alert_enabled = true;

-- Step 1b: Extend saved_opportunities with tracking columns
ALTER TABLE saved_opportunities
    ADD COLUMN notice_id VARCHAR(100),
    ADD COLUMN solicitation_number VARCHAR(100),
    ADD COLUMN notes TEXT,
    ADD COLUMN last_notified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Backfill notice_id and solicitation_number from opportunities
UPDATE saved_opportunities so
SET notice_id = o.notice_id,
    solicitation_number = o.solicitation_number
FROM opportunities o
WHERE o.id = so.opportunity_id;

-- Now enforce NOT NULL on notice_id (all rows should have it after backfill)
ALTER TABLE saved_opportunities ALTER COLUMN notice_id SET NOT NULL;

CREATE INDEX idx_saved_opportunities_notice_id ON saved_opportunities (notice_id);
CREATE INDEX idx_saved_opportunities_sol_num ON saved_opportunities (solicitation_number) WHERE solicitation_number IS NOT NULL;
CREATE INDEX idx_saved_opportunities_user_created ON saved_opportunities (user_id, created_at DESC);

-- Step 1c: Create user_updates table
-- Retention policy: purge rows older than 90 days (via scheduled job or app logic)
CREATE TABLE user_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    update_type VARCHAR(50) NOT NULL,
    source_id INTEGER,
    opportunity_ids INTEGER[],
    summary TEXT NOT NULL,
    details JSONB,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_updates_feed ON user_updates (user_id, is_read, created_at DESC);
CREATE INDEX idx_user_updates_source ON user_updates (source_id) WHERE source_id IS NOT NULL;

-- Step 1d: Create alert_jobs lock table
CREATE TABLE alert_jobs (
    job_name TEXT PRIMARY KEY,
    last_run_at TIMESTAMPTZ,
    locked_at TIMESTAMPTZ,
    locked_by TEXT
);

INSERT INTO alert_jobs (job_name) VALUES
    ('saved_search_alerts'),
    ('opportunity_change_alerts');
