-- 011_lab_ftth.sql : persisted connection-lab evidence + FTTH depth
CREATE TABLE IF NOT EXISTS lab_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
  vendor TEXT NOT NULL, family TEXT NOT NULL, connection_type TEXT NOT NULL,
  host TEXT NOT NULL,
  healthy BOOLEAN NOT NULL, latency_ms BIGINT NOT NULL,
  device_info JSONB, capabilities JSONB, probes JSONB,
  error TEXT, tested_by TEXT, tested_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_labruns_org ON lab_runs(org_id, tested_at DESC);

-- FTTH depth on existing tables
ALTER TABLE onus ADD COLUMN IF NOT EXISTS tx_power FLOAT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS temperature FLOAT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS distance_m INT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS loid TEXT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS line_profile TEXT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS service_profile TEXT;
ALTER TABLE onus ADD COLUMN IF NOT EXISTS vlan INT;
ALTER TABLE olt_devices ADD COLUMN IF NOT EXISTS shelf INT DEFAULT 0;
ALTER TABLE olt_devices ADD COLUMN IF NOT EXISTS slot INT DEFAULT 0;
ALTER TABLE olt_devices ADD COLUMN IF NOT EXISTS pon_count INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS splitters (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  olt_id UUID REFERENCES olt_devices(id) ON DELETE SET NULL,
  name TEXT NOT NULL, ratio TEXT NOT NULL, pon_port TEXT,
  lat DOUBLE PRECISION, lng DOUBLE PRECISION,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS onu_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id),
  vendor TEXT NOT NULL, name TEXT NOT NULL,
  down_mbps INT NOT NULL, up_mbps INT NOT NULL,
  vlan INT, mode TEXT DEFAULT 'bridge',
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(org_id, vendor, name)
);
