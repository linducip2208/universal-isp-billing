# Testing

Levels: unit (domain logic), HTTP (auth/throttle/RBAC/503 honesty/revocation),
repository (live PG: lists, NOC aggregates, economics, tenant isolation —
gate `TEST_DATABASE_URL`), connector contract (scripted fakes: MikroTik,
SNMP agent), protocol (RADIUS codec/dedup/usage, SNMPv3 MAC), E2E (env-gated:
MIKROTIK_E2E/RADIUS_E2E/SNMP_E2E, all SKIP without hardware), live infra
(Memurai: lock/streams, gate `TEST_REDIS_ADDR`), benchmarks (`bench_test.go`,
recorded in PERFORMANCE.md), race (`make test-race`, CI/gcc required).
Frontend: production build gate (no unit framework yet — PLANNED).
