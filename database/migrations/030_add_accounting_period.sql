-- Accounting period: the tax period (VAT return month) this document is claimed in.
-- Distinct from invoice_year/month/day which is the date printed on the document.
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS accounting_year  INT,
  ADD COLUMN IF NOT EXISTS accounting_month INT;

-- Backfill existing records: use the month the record was created (upload month).
UPDATE invoices
SET accounting_year  = EXTRACT(YEAR  FROM created_at)::int,
    accounting_month = EXTRACT(MONTH FROM created_at)::int
WHERE accounting_year IS NULL;

CREATE INDEX IF NOT EXISTS idx_invoices_accounting_period
  ON invoices (tenant_id, accounting_year, accounting_month);
