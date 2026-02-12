CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    workos_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(320) NOT NULL,
    first_name VARCHAR(200),
    last_name VARCHAR(200),
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
