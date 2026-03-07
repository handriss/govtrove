ALTER TABLE mcp_usage
    ADD COLUMN request_params JSONB,
    ADD COLUMN result_count INTEGER,
    ADD COLUMN user_email TEXT;
