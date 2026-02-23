DROP TABLE IF EXISTS alert_jobs;
DROP TABLE IF EXISTS user_updates;

DROP INDEX IF EXISTS idx_saved_opportunities_user_created;
DROP INDEX IF EXISTS idx_saved_opportunities_sol_num;
DROP INDEX IF EXISTS idx_saved_opportunities_notice_id;

ALTER TABLE saved_opportunities
    DROP COLUMN IF EXISTS notice_id,
    DROP COLUMN IF EXISTS solicitation_number,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS last_notified_at,
    DROP COLUMN IF EXISTS updated_at;

DROP INDEX IF EXISTS idx_saved_searches_alert_enabled;

ALTER TABLE saved_searches
    DROP COLUMN IF EXISTS alert_enabled,
    DROP COLUMN IF EXISTS last_checked_at,
    DROP COLUMN IF EXISTS last_match_count,
    DROP COLUMN IF EXISTS total_result_count;
