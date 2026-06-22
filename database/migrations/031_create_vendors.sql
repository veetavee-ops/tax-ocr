-- Vendor registry: canonical vendor info keyed by Thai tax ID (13 digits).
-- tax_id is globally unique (issued by Revenue Department — same across all tenants).
-- Once verified, all future invoices with the same tax_id auto-link here.
CREATE TABLE IF NOT EXISTS vendors (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  tax_id       VARCHAR(13)  NOT NULL,
  name         VARCHAR(255),
  address      TEXT,
  branch_code  VARCHAR(10),
  branch_name  VARCHAR(100),
  phone        VARCHAR(20),
  verified     BOOLEAN      NOT NULL DEFAULT FALSE,
  verified_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
  verified_at  TIMESTAMPTZ,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_vendors_tax_id UNIQUE (tax_id)
);

CREATE INDEX IF NOT EXISTS idx_vendors_tax_id ON vendors (tax_id);

-- Link invoices to the vendor registry.
-- NULL = vendor not yet looked up / tax_id missing from OCR.
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_invoices_vendor_id ON invoices (vendor_id) WHERE vendor_id IS NOT NULL;
