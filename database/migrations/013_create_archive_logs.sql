CREATE TABLE IF NOT EXISTS archive_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id),
    entity_type  VARCHAR(50) NOT NULL,
    entity_id    UUID        NOT NULL,
    archived_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archive_path TEXT        NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'archived' CHECK (status IN ('archived', 'restored')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_archive_logs_tenant_id   ON archive_logs (tenant_id);
CREATE INDEX idx_archive_logs_entity_type ON archive_logs (entity_type);
CREATE INDEX idx_archive_logs_entity_id   ON archive_logs (entity_id);
CREATE INDEX idx_archive_logs_status      ON archive_logs (status);
