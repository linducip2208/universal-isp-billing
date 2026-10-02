-- 005_partitioning.sql : high-volume tables use declarative RANGE partitioning
-- on recorded_at/sampled_at/created_at so millions of accounting/telemetry
-- rows stay queryable and droppable by month.
--
-- Fresh installs: run this INSTEAD of the plain CREATEs in 001_init.sql for
-- the tables below. Existing installs: backfill with pg_partman or the
-- documented procedure in docs/PERFORMANCE.md (do not run blindly on live data).

-- Example (radius_accounting). Repeat the pattern for traffic_samples,
-- interface_samples, events, audit_logs.
CREATE TABLE IF NOT EXISTS radius_accounting_p (
  id BIGSERIAL,
  username TEXT NOT NULL,
  acct_session_id TEXT NOT NULL,
  acct_status_type TEXT NOT NULL,
  input_octets BIGINT DEFAULT 0,
  output_octets BIGINT DEFAULT 0,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, recorded_at)
) PARTITION BY RANGE (recorded_at);

-- Operator creates monthly partitions, e.g.:
--   CREATE TABLE radius_accounting_2026_10 PARTITION OF radius_accounting_p
--     FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
--   CREATE INDEX ON radius_accounting_2026_10 (acct_session_id, recorded_at);
-- Retention: DETACH + DROP old monthly partitions (documented in OPERATIONS.md).
