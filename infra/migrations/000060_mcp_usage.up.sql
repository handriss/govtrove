CREATE TABLE mcp_usage (
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    tool_name  TEXT NOT NULL,
    called_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    latency_ms INTEGER
);

CREATE INDEX idx_mcp_usage_user_called ON mcp_usage (user_id, called_at DESC);
