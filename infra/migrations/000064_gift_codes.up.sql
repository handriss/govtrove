CREATE TABLE gift_codes (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    campaign_name TEXT NOT NULL,
    duration_days INT NOT NULL DEFAULT 365,
    max_redemptions INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ
);

CREATE TABLE gift_code_redemptions (
    id SERIAL PRIMARY KEY,
    gift_code_id INT NOT NULL REFERENCES gift_codes(id),
    user_id INT NOT NULL REFERENCES users(id),
    granted_until TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(gift_code_id, user_id)
);

ALTER TABLE users ADD COLUMN gift_expires_at TIMESTAMPTZ;
