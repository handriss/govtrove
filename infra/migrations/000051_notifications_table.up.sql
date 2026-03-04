CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    update_type VARCHAR(50) NOT NULL,
    source_id   INTEGER,
    group_key   VARCHAR(100),
    details     JSONB NOT NULL,
    is_read     BOOLEAN NOT NULL DEFAULT false,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '90 days',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notification_type CHECK (update_type IN ('search_matches', 'opportunity_update'))
);

CREATE INDEX idx_notifications_feed ON notifications (user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_source ON notifications (source_id) WHERE source_id IS NOT NULL;
CREATE UNIQUE INDEX idx_notifications_group_key ON notifications (group_key) WHERE group_key IS NOT NULL;
CREATE INDEX idx_notifications_expires ON notifications (expires_at);

INSERT INTO alert_jobs (job_name) VALUES
    ('search_notifications'),
    ('opportunity_notifications');
