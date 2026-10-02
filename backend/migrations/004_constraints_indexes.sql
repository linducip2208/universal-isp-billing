-- 004_constraints_indexes.sql : harden schema (FKs, indexes, checks, numbering, journal)
ALTER TABLE subscriptions ADD CONSTRAINT fk_sub_device FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL;
ALTER TABLE payments ADD CONSTRAINT fk_pay_invoice FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_invoices_customer ON invoices(customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_due ON invoices(due_at) WHERE status IN ('open','overdue');
CREATE INDEX IF NOT EXISTS idx_payments_invoice ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_devices_org_status ON devices(org_id, status);
CREATE INDEX IF NOT EXISTS idx_subs_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON radius_sessions(username);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON network_jobs(status);
CREATE INDEX IF NOT EXISTS idx_traffic_device ON traffic_samples(device_id, sampled_at DESC);
CREATE INDEX IF NOT EXISTS idx_iface_device ON interface_samples(device_id, ifname, sampled_at DESC);

ALTER TABLE invoices ADD CONSTRAINT chk_invoice_totals CHECK (total_cents >= 0 AND paid_cents >= 0);
ALTER TABLE payments ADD CONSTRAINT chk_payment_amount CHECK (amount_cents > 0);
ALTER TABLE packages ADD CONSTRAINT chk_package_price CHECK (price_cents >= 0);

-- Invoice numbering sequences (org-scoped, transactional; see billing.SQLSequence)
CREATE TABLE IF NOT EXISTS invoice_sequences (
  org_id UUID NOT NULL REFERENCES organizations(id),
  period TEXT NOT NULL,
  seq BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (org_id, period)
);

-- Accounting journal (balanced entries; idempotency unique per org)
CREATE TABLE IF NOT EXISTS journal_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  invoice_id UUID REFERENCES invoices(id),
  payment_id UUID REFERENCES payments(id),
  memo TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(org_id, idempotency_key)
);
CREATE TABLE IF NOT EXISTS journal_lines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
  account TEXT NOT NULL,
  debit_cents BIGINT NOT NULL DEFAULT 0 CHECK (debit_cents >= 0),
  credit_cents BIGINT NOT NULL DEFAULT 0 CHECK (credit_cents >= 0),
  CHECK (NOT (debit_cents > 0 AND credit_cents > 0))
);
