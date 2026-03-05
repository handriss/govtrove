ALTER TABLE users
    ADD COLUMN subscription_id TEXT,
    ADD COLUMN subscription_status TEXT,
    ADD COLUMN cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN current_period_end TIMESTAMPTZ;

CREATE TABLE stripe_webhook_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stripe_webhook_events_processed_at ON stripe_webhook_events (processed_at);
