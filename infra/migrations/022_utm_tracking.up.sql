CREATE TABLE utm_visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    landing_page TEXT NOT NULL,
    origin TEXT,
    referrer TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_utm_visits_campaign ON utm_visits(utm_campaign);
CREATE INDEX idx_utm_visits_created_at ON utm_visits(created_at);

ALTER TABLE search_events ADD COLUMN utm_campaign TEXT;
CREATE INDEX idx_search_events_utm_campaign ON search_events(utm_campaign) WHERE utm_campaign IS NOT NULL;
