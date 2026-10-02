-- 007_operations.sql : incidents, field service, contracts/SLA, config management
CREATE TABLE IF NOT EXISTS incidents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  title TEXT NOT NULL, severity TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open',
  scope_key TEXT NOT NULL, alert_ids TEXT[] DEFAULT '{}',
  impacted_services TEXT[] DEFAULT '{}', impacted_subs INT DEFAULT 0,
  root_cause TEXT, postmortem TEXT, assignee TEXT,
  opened_at TIMESTAMPTZ DEFAULT now(), acked_at TIMESTAMPTZ, resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_incidents_org_status ON incidents(org_id, status);

CREATE TABLE IF NOT EXISTS tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  customer_id UUID REFERENCES customers(id), subscription_id UUID REFERENCES subscriptions(id),
  device_id UUID REFERENCES devices(id),
  subject TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open', priority TEXT DEFAULT 'normal',
  created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tickets_org_status ON tickets(org_id, status);

CREATE TABLE IF NOT EXISTS work_orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  ticket_id UUID REFERENCES tickets(id),
  technician_id UUID, kind TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open',
  scheduled_for TIMESTAMPTZ, notes TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS technicians (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  name TEXT NOT NULL, phone TEXT, area TEXT
);

CREATE TABLE IF NOT EXISTS spare_parts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  sku TEXT NOT NULL, name TEXT NOT NULL, quantity INT NOT NULL DEFAULT 0,
  UNIQUE(org_id, sku)
);

CREATE TABLE IF NOT EXISTS contracts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  customer_id UUID NOT NULL REFERENCES customers(id),
  kind TEXT NOT NULL, sla_id UUID, starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL, mrc_cents BIGINT NOT NULL DEFAULT 0 CHECK (mrc_cents >= 0),
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS sla_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  name TEXT NOT NULL, uptime_pct DOUBLE PRECISION NOT NULL,
  response_sla INTERVAL NOT NULL DEFAULT '4 hours',
  resolution_sla INTERVAL NOT NULL DEFAULT '24 hours',
  maintenance_exclusion BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS config_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  version INT NOT NULL, body TEXT NOT NULL,
  taken_by TEXT, taken_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(device_id, version)
);

CREATE TABLE IF NOT EXISTS changes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  summary TEXT NOT NULL, diff TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'proposed',
  proposed_by TEXT, approved_by TEXT, scheduled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_changes_device ON changes(device_id, status);

-- Tenant-scope the session table (older installs lack org_id).
ALTER TABLE radius_sessions ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id);
CREATE INDEX IF NOT EXISTS idx_sessions_org_user ON radius_sessions(org_id, username);
