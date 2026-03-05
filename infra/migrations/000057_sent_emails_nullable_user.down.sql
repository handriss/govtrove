UPDATE sent_emails SET user_id = 0 WHERE user_id IS NULL;
ALTER TABLE sent_emails ALTER COLUMN user_id SET NOT NULL;
