CREATE TABLE sent_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_email VARCHAR(320) NOT NULL,
    email_type VARCHAR(50) NOT NULL,
    template_name VARCHAR(100) NOT NULL,
    template_data JSONB,
    subject TEXT NOT NULL,
    resend_message_id TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'sent',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sent_emails_user_id ON sent_emails(user_id);
CREATE INDEX idx_sent_emails_status ON sent_emails(status);
CREATE INDEX idx_sent_emails_created_at ON sent_emails(created_at DESC);
CREATE INDEX idx_sent_emails_resend_message_id ON sent_emails(resend_message_id);
