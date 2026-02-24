ALTER TABLE pipeline.samgov_requests ADD COLUMN api_key_hash VARCHAR(16);

CREATE TABLE pipeline.api_keys (
    key_hash        VARCHAR(16) PRIMARY KEY,
    email           TEXT NOT NULL,
    daily_limit     INT NOT NULL DEFAULT 1000,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

INSERT INTO pipeline.api_keys (key_hash, email, daily_limit, created_at, expires_at) VALUES
    ('85780067', 'andrew@govtrove.com', 1000, NOW(), NOW() + INTERVAL '90 days'),
    ('560d65d4', 'andrashinkel+samgov@gmail.com', 10, NOW(), NOW() + INTERVAL '90 days');
