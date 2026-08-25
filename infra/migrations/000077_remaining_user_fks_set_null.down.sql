-- Rows with a NULL user_id (from a completed account deletion) must be removed
-- before NOT NULL can be restored.
DELETE FROM mcp_usage WHERE user_id IS NULL;
ALTER TABLE mcp_usage DROP CONSTRAINT IF EXISTS mcp_usage_user_id_fkey;
ALTER TABLE mcp_usage ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE mcp_usage ADD CONSTRAINT mcp_usage_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

DELETE FROM invite_link_redemptions WHERE user_id IS NULL;
ALTER TABLE invite_link_redemptions DROP CONSTRAINT IF EXISTS invite_link_redemptions_user_id_fkey;
ALTER TABLE invite_link_redemptions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE invite_link_redemptions ADD CONSTRAINT invite_link_redemptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

DELETE FROM gift_code_redemptions WHERE user_id IS NULL;
ALTER TABLE gift_code_redemptions DROP CONSTRAINT IF EXISTS gift_code_redemptions_user_id_fkey;
ALTER TABLE gift_code_redemptions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE gift_code_redemptions ADD CONSTRAINT gift_code_redemptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
