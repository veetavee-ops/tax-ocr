CREATE TABLE IF NOT EXISTS archive_policies (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    active_days  INT         NOT NULL DEFAULT 90,
    archive_days INT         NOT NULL DEFAULT 365,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id)
);

CREATE INDEX idx_archive_policies_tenant_id ON archive_policies (tenant_id);
