CREATE TABLE IF NOT EXISTS classification_rules (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID           NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    keyword    VARCHAR(255)   NOT NULL,
    asset_type VARCHAR(20)    NOT NULL CHECK (asset_type IN ('asset', 'expense')),
    source     VARCHAR(20)    NOT NULL DEFAULT 'human' CHECK (source IN ('ai', 'human')),
    confidence NUMERIC(5, 4)  NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, keyword)
);

CREATE INDEX idx_classification_rules_tenant_id  ON classification_rules (tenant_id);
CREATE INDEX idx_classification_rules_keyword     ON classification_rules (keyword);
CREATE INDEX idx_classification_rules_asset_type  ON classification_rules (asset_type);
