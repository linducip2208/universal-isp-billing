-- 009_list_indexes.sql : composite (org_id, created_at) for every paginated
-- list endpoint (EXPLAIN showed seq scans on subscriptions without it).
CREATE INDEX IF NOT EXISTS idx_subs_org_created ON subscriptions(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_org_created ON invoices(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_devices_org_created ON devices(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_org_created ON alerts(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_org_created ON events(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_org_created ON network_jobs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_org_created ON audit_logs(org_id, created_at DESC);
