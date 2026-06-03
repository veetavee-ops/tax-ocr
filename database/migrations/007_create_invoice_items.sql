CREATE TABLE IF NOT EXISTS invoice_items (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID           NOT NULL REFERENCES tenants (id),
    branch_id     UUID           NOT NULL REFERENCES branches (id),
    invoice_id    UUID           NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    description   TEXT           NOT NULL,
    quantity      NUMERIC(15, 4) NOT NULL DEFAULT 1,
    unit_price    NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total_price   NUMERIC(15, 2) NOT NULL DEFAULT 0,
    asset_type    VARCHAR(20)    NOT NULL DEFAULT 'pending' CHECK (asset_type IN ('asset', 'expense', 'pending')),
    classified_by VARCHAR(20)    NOT NULL DEFAULT 'rule'   CHECK (classified_by IN ('rule', 'ai', 'human')),
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoice_items_tenant_id  ON invoice_items (tenant_id);
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items (invoice_id);
CREATE INDEX idx_invoice_items_asset_type ON invoice_items (asset_type);
