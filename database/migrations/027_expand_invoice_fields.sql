-- Expand invoices with document classification, seller/buyer full info, and summary fields
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS doc_type              VARCHAR(50)    DEFAULT 'tax_invoice',
  ADD COLUMN IF NOT EXISTS vat_inclusive         BOOLEAN        DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS vat_rate              DECIMAL(5,2)   DEFAULT 7.00,
  ADD COLUMN IF NOT EXISTS vendor_address        TEXT,
  ADD COLUMN IF NOT EXISTS vendor_branch_code    VARCHAR(50),
  ADD COLUMN IF NOT EXISTS buyer_name            VARCHAR(255),
  ADD COLUMN IF NOT EXISTS buyer_tax_id          VARCHAR(13),
  ADD COLUMN IF NOT EXISTS buyer_address         TEXT,
  ADD COLUMN IF NOT EXISTS buyer_branch_code     VARCHAR(50),
  ADD COLUMN IF NOT EXISTS vat_exempt_amount     DECIMAL(15,2)  DEFAULT 0,
  ADD COLUMN IF NOT EXISTS vat_inclusive_subtotal DECIMAL(15,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS discount_amount       DECIMAL(15,2)  DEFAULT 0;
