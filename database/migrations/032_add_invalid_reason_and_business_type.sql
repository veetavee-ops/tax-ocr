-- Add invalid_reason to invoices for legal compliance failures (distinct from OCR conflict)
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS invalid_reason TEXT;

-- Add business_type to tenants: trading / service / construction
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS business_type VARCHAR(20) NOT NULL DEFAULT 'service';
