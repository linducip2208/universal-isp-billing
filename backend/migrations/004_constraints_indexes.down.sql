-- 004_constraints_indexes.down.sql
DROP TABLE IF EXISTS journal_lines, journal_entries, invoice_sequences;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS fk_sub_device;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS fk_pay_invoice;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS chk_invoice_totals;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS chk_payment_amount;
ALTER TABLE packages DROP CONSTRAINT IF EXISTS chk_package_price;
DROP INDEX IF EXISTS idx_invoices_customer, idx_invoices_due, idx_payments_invoice,
  idx_devices_org_status, idx_subs_status, idx_sessions_user, idx_events_type,
  idx_audit_created, idx_alerts_status, idx_jobs_status, idx_traffic_device, idx_iface_device;
