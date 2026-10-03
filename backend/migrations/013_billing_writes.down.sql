-- 013_billing_writes.down.sql
DROP TABLE IF EXISTS vouchers;
DROP TABLE IF EXISTS invoice_idem;
DROP INDEX IF EXISTS idx_invoices_number;
ALTER TABLE invoices DROP COLUMN IF EXISTS number;
