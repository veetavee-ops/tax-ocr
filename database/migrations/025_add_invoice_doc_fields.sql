ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS invoice_doc_no TEXT,
  ADD COLUMN IF NOT EXISTS invoice_date   TEXT;
