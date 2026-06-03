CREATE TABLE IF NOT EXISTS messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    tenant_id       UUID        NOT NULL REFERENCES tenants (id),
    sender_type     VARCHAR(20) NOT NULL CHECK (sender_type IN ('customer', 'admin', 'bot')),
    sender_id       UUID,
    message_type    VARCHAR(20) NOT NULL CHECK (message_type IN ('text', 'image', 'file', 'sticker')),
    content         TEXT        NOT NULL,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id);
CREATE INDEX idx_messages_tenant_id       ON messages (tenant_id);
CREATE INDEX idx_messages_sender_id       ON messages (sender_id);
CREATE INDEX idx_messages_created_at      ON messages (created_at DESC);
