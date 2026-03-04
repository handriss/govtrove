CREATE TABLE email_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    search_alerts BOOLEAN NOT NULL DEFAULT true,
    opportunity_alerts BOOLEAN NOT NULL DEFAULT true,
    unsubscribed_at TIMESTAMPTZ,
    unsubscribe_reason VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
