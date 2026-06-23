-- 034: tenant soft delete + suspend
ALTER TABLE tenants
  ADD COLUMN deleted_at        TIMESTAMPTZ,
  ADD COLUMN suspended_at      TIMESTAMPTZ,
  ADD COLUMN suspension_reason TEXT;
