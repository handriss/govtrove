-- Completes the account-deletion fix started in 000076. These three tables also
-- referenced users with NO ACTION, so deletion still failed for anyone who had
-- used MCP or redeemed a code.
--
-- SET NULL rather than CASCADE for all three: gift_codes/invite_links enforce
-- max_redemptions by COUNT(*) over these tables with no counter column, so
-- deleting a redemption row would silently hand a redemption slot back.
-- user_id is NOT NULL on all three, so the constraint has to go first.
ALTER TABLE mcp_usage ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE mcp_usage DROP CONSTRAINT IF EXISTS mcp_usage_user_id_fkey;
ALTER TABLE mcp_usage ADD CONSTRAINT mcp_usage_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE invite_link_redemptions ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE invite_link_redemptions DROP CONSTRAINT IF EXISTS invite_link_redemptions_user_id_fkey;
ALTER TABLE invite_link_redemptions ADD CONSTRAINT invite_link_redemptions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE gift_code_redemptions ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE gift_code_redemptions DROP CONSTRAINT IF EXISTS gift_code_redemptions_user_id_fkey;
ALTER TABLE gift_code_redemptions ADD CONSTRAINT gift_code_redemptions_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
