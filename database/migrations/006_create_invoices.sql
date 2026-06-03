CREATE TABLE IF NOT EXISTS invoices (
    id                UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID           NOT NULL REFERENCES tenants (id),
    branch_id         UUID           NOT NULL REFERENCES branches (id),
    document_import_id UUID          REFERENCES document_imports (id),
    file_path         TEXT           NOT NULL,
    file_hash         VARCHAR(64)    NOT NULL,
    vendor_tax_id     VARCHAR(13),
    total_before_vat  NUMERIC(15, 2) NOT NULL DEFAULT 0,
    vat_amount        NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total_amount      NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status            VARCHAR(20)    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verified', 'conflict')),
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_tenant_id      ON invoices (tenant_id);
CREATE INDEX idx_invoices_branch_id      ON invoices (branch_id);
CREATE INDEX idx_invoices_vendor_tax_id  ON invoices (vendor_tax_id);
CREATE INDEX idx_invoices_status         ON invoices (status);
CREATE INDEX idx_invoices_file_hash      ON invoices (file_hash);
