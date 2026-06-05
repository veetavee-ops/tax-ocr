-- Expand invoice_items with product code, unit, and per-line discount
ALTER TABLE invoice_items
  ADD COLUMN IF NOT EXISTS product_code  VARCHAR(100),
  ADD COLUMN IF NOT EXISTS unit          VARCHAR(50),
  ADD COLUMN IF NOT EXISTS discount      DECIMAL(15,2) DEFAULT 0;
