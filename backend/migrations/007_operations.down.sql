-- 007_operations.down.sql (radius_sessions.org_id kept — dropping loses tenant scope)
DROP TABLE IF EXISTS changes, config_snapshots, sla_policies, contracts CASCADE;
DROP TABLE IF EXISTS spare_parts, technicians, work_orders, tickets CASCADE;
DROP TABLE IF EXISTS incidents CASCADE;
