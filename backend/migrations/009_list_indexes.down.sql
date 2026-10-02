-- 009_list_indexes.down.sql
DROP INDEX IF EXISTS idx_subs_org_created, idx_invoices_org_created, idx_devices_org_created,
  idx_alerts_org_created, idx_events_org_created, idx_jobs_org_created, idx_audit_org_created;
