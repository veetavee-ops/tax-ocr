CREATE TABLE IF NOT EXISTS document_imports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL REFERENCES tenants (id),
    branch_id       UUID         NOT NULL REFERENCES branches (id),
    user_id         UUID         NOT NULL REFERENCES users (id),
    source_type     VARCHAR(20)  NOT NULL CHECK (source_type IN ('camera', 'upload', 'zip', 'gdrive', 'onedrive')),
    source_url      TEXT,
    total_files     INT          NOT NULL DEFAULT 0,
    processed_files INT          NOT NULL DEFAULT 0,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_imports_tenant_id ON document_imports (tenant_id);
CREATE INDEX idx_document_imports_branch_id ON document_imports (branch_id);
CREATE INDEX idx_document_imports_user_id   ON document_imports (user_id);
CREATE INDEX idx_document_imports_status    ON document_imports (status);
