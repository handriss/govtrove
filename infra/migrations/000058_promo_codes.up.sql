CREATE TABLE promo_codes (
    id              SERIAL PRIMARY KEY,
    code            TEXT NOT NULL UNIQUE,
    stripe_promo_id TEXT NOT NULL,
    for_user_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    redeemed_by     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    redeemed_at     TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ
);

CREATE INDEX idx_promo_codes_for_user ON promo_codes (for_user_id);
CREATE INDEX idx_promo_codes_code ON promo_codes (code);
