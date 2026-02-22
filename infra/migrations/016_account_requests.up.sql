CREATE TABLE account_requests (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    request_type VARCHAR(20) NOT NULL CHECK (request_type IN ('data_export', 'account_deletion')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_account_requests_user_id ON account_requests(user_id);
