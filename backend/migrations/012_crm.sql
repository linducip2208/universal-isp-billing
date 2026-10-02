-- 012_crm.sql : lead pipeline
CREATE TABLE IF NOT EXISTS crm_leads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  name TEXT NOT NULL, phone TEXT, address TEXT,
  stage TEXT NOT NULL DEFAULT 'lead',
  quoted_cents BIGINT NOT NULL DEFAULT 0 CHECK (quoted_cents >= 0),
  customer_id UUID REFERENCES customers(id),
  created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_leads_org_stage ON crm_leads(org_id, stage);
