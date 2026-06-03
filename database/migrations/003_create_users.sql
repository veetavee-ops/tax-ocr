CREATE TABLE IF NOT EXISTS users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    email        VARCHAR(255),
    phone        VARCHAR(20),
    line_user_id VARCHAR(100),
    role         VARCHAR(20)  NOT NULL CHECK (role IN ('admin', 'staff')),
    status       VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_tenant_id    ON users (tenant_id);
CREATE INDEX idx_users_email        ON users (email);
CREATE INDEX idx_users_line_user_id ON users (line_user_id);
CREATE INDEX idx_users_status       ON users (status);
