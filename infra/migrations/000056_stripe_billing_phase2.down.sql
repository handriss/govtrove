DROP TABLE IF EXISTS stripe_webhook_events;
ALTER TABLE users
    DROP COLUMN IF EXISTS subscription_id,
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS cancel_at_period_end,
    DROP COLUMN IF EXISTS current_period_end;
