-- Store registered address for report headers (รายงานภาษีซื้อ ม.87/1, 50 ทวิ, e-Tax XML)
-- NOT used in OCR buyer validation — only for reporting/billing modules
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS address TEXT;

ALTER TABLE branches ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE branches ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
