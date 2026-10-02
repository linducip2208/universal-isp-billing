# Connection Lab

`POST /api/v1/lab/test` (RBAC `network:write`) or `ispctl device test`:
vendor → family → protocol → host/port → auth → TLS → timeout → test →
discover (info, interfaces, per-capability probes) → latency/errors/last
result. Credentials are request-scoped, never logged or stored. Results map
to `devices`/`device_capabilities` rows for audit. Frontend: /lab page.
