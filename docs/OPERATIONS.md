# Operations / DR / Scaling / Troubleshooting / Network automation

- Operations: `ispctl doctor` (config/deps, no secrets), `ispctl health`,
  systemd units, nginx example, migration chain 001–005 (`ispctl migrate`
  procedure), log via journald, `system_settings` for runtime config.
- DR: Postgres `pg_dump` backup/restore; config backup via MikroTik
  `ExportConfig`; secret backup = encrypted vault file + offline key copy;
  migration rollback = down-migrations per file (PLANNED automation);
  failed provisioning recovery = DLQ + operator retry + drift re-plan.
- Scaling: vertical first; workers scale horizontally once Redis transport
  lands; read replicas for reports (PLANNED).
- Troubleshooting: request IDs correlate logs→audit; lab probes isolate
  device vs credential vs network faults; 503s name the missing dependency.
- Network automation doc: see IPAM.md + automation engine + drift plans.
