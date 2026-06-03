CREATE TABLE IF NOT EXISTS tenant_storage_config (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID           NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    storage_type        VARCHAR(20)    NOT NULL CHECK (storage_type IN ('gdrive', 'onedrive', 'both')),
    gdrive_folder_id    TEXT,
    gdrive_folder_url   TEXT,
    onedrive_folder_id  TEXT,
    onedrive_folder_url TEXT,
    owned_by            VARCHAR(20)    NOT NULL DEFAULT 'us' CHECK (owned_by IN ('tenant', 'us')),
    billing_type        VARCHAR(20)    NOT NULL DEFAULT 'included' CHECK (billing_type IN ('included', 'charged')),
    monthly_fee         NUMERIC(10, 2) NOT NULL DEFAULT 0,
    status              VARCHAR(20)    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id)
);

CREATE INDEX idx_tenant_storage_config_tenant_id ON tenant_storage_config (tenant_id);
