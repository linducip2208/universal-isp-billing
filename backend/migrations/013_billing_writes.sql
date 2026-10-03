-- 013_billing_writes.sql : invoice numbers/idempotency, vouchers
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS number TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_number ON invoices(number) WHERE number IS NOT NULL;

CREATE TABLE IF NOT EXISTS invoice_idem (
  org_id UUID NOT NULL REFERENCES organizations(id),
  idem_key TEXT NOT NULL,
  invoice_id UUID NOT NULL REFERENCES invoices(id),
  created_at TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (org_id, idem_key)
);

CREATE TABLE IF NOT EXISTS vouchers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  code TEXT NOT NULL,
  package_id UUID REFERENCES packages(id),
  duration_hours INT NOT NULL DEFAULT 24 CHECK (duration_hours > 0),
  status TEXT NOT NULL DEFAULT 'active',
  used_by TEXT,
  used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(org_id, code)
);
CREATE INDEX IF NOT EXISTS idx_vouchers_org_status ON vouchers(org_id, status);
