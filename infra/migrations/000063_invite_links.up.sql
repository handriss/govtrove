CREATE TABLE invite_links (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    stripe_promo_id TEXT NOT NULL,
    campaign_name TEXT NOT NULL,
    max_redemptions INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ
);

CREATE TABLE invite_link_redemptions (
    id SERIAL PRIMARY KEY,
    invite_link_id INT NOT NULL REFERENCES invite_links(id),
    user_id INT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(invite_link_id, user_id)
);
