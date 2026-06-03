CREATE TABLE IF NOT EXISTS hitl_queue (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id),
    invoice_item_id UUID        NOT NULL REFERENCES invoice_items (id) ON DELETE CASCADE,
    reason          TEXT        NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved')),
    resolved_by     UUID        REFERENCES users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hitl_queue_tenant_id       ON hitl_queue (tenant_id);
CREATE INDEX idx_hitl_queue_status          ON hitl_queue (status);
CREATE INDEX idx_hitl_queue_invoice_item_id ON hitl_queue (invoice_item_id);
