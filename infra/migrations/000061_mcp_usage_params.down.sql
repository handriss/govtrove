ALTER TABLE mcp_usage
    DROP COLUMN IF EXISTS request_params,
    DROP COLUMN IF EXISTS result_count,
    DROP COLUMN IF EXISTS user_email;
