CREATE TABLE IF NOT EXISTS conversations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id),
    branch_id    UUID        NOT NULL REFERENCES branches (id),
    user_id      UUID        REFERENCES users (id),
    channel      VARCHAR(20) NOT NULL CHECK (channel IN ('line_oa', 'liff')),
    line_user_id VARCHAR(100),
    status       VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_tenant_id    ON conversations (tenant_id);
CREATE INDEX idx_conversations_branch_id    ON conversations (branch_id);
CREATE INDEX idx_conversations_user_id      ON conversations (user_id);
CREATE INDEX idx_conversations_line_user_id ON conversations (line_user_id);
CREATE INDEX idx_conversations_status       ON conversations (status);
