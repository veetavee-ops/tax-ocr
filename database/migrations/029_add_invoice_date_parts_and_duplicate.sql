-- Add calendar fields for accounting period grouping
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS invoice_year  INT,
  ADD COLUMN IF NOT EXISTS invoice_month INT,
  ADD COLUMN IF NOT EXISTS invoice_day   INT,
  ADD COLUMN IF NOT EXISTS duplicate_of  UUID REFERENCES invoices(id);

-- Mark existing duplicate invoices (same tenant+vendor+doc_no) before creating unique index.
-- Keep the earliest-created as the original; mark later ones as duplicates.
UPDATE invoices AS dup
SET duplicate_of = orig.id
FROM (
  SELECT DISTINCT ON (tenant_id, vendor_tax_id, invoice_doc_no)
    id,
    tenant_id,
    vendor_tax_id,
    invoice_doc_no
  FROM invoices
  WHERE vendor_tax_id IS NOT NULL
    AND invoice_doc_no IS NOT NULL
  ORDER BY tenant_id, vendor_tax_id, invoice_doc_no, created_at ASC
) AS orig
WHERE dup.tenant_id      = orig.tenant_id
  AND dup.vendor_tax_id  = orig.vendor_tax_id
  AND dup.invoice_doc_no = orig.invoice_doc_no
  AND dup.id            != orig.id
  AND dup.duplicate_of  IS NULL;

-- Prevent same vendor from having the same invoice_doc_no within a tenant.
-- Partial index only covers "original" invoices (duplicate_of IS NULL).
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoices_vendor_doc_no
  ON invoices (tenant_id, vendor_tax_id, invoice_doc_no)
  WHERE vendor_tax_id IS NOT NULL
    AND invoice_doc_no IS NOT NULL
    AND duplicate_of IS NULL;

-- Support fast grouping/filtering by accounting period
CREATE INDEX IF NOT EXISTS idx_invoices_year_month
  ON invoices (tenant_id, invoice_year, invoice_month);

-- Support finding duplicates quickly
CREATE INDEX IF NOT EXISTS idx_invoices_duplicate_of
  ON invoices (duplicate_of)
  WHERE duplicate_of IS NOT NULL;
