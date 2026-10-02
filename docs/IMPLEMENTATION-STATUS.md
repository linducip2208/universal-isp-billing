# IMPLEMENTATION STATUS — Universal ISP Network Operating Platform

Generated from code inspection + test runs. Code is the source of truth.
Status vocabulary: ARCHITECTURE COMPLETE | FEATURE COMPLETE |
PROTOCOL IMPLEMENTED | REAL DEVICE VERIFIED | PRODUCTION READY |
REQUIRES EXTERNAL CREDENTIALS | PARTIAL | PLANNED.

## CURRENT VERSION
- Backend: Go modular monolith (`backend/`, module `github.com/universal-isp/platform`).
- Deps: `github.com/lib/pq` (postgres driver), `golang.org/x/crypto` (PBKDF2). No cgo.
- Frontend: React 18 + TS + Vite 5, HashRouter, i18n en/id, dark mode.
- DB: PostgreSQL migrations 001–005. Redis optional (cache client stdlib RESP).

## ARCHITECTURE STATUS — ARCHITECTURE COMPLETE
Clean modular monolith: `cmd/{server,ispctl}`, `internal/{auth,rbac,store,
billing,provisioning,automation,monitoring,alerts,topology,scheduler,cache,
reports,syslog,tr069,webhooks,apikey,ipam,snmp,radius,devices,customers,
subscriptions,packages,connectors/{sdk,registry,all,lab,mikrotik,generic,cloud,routers,olt},
httpapi,middleware,events,jobs,workers,health,security,i18n,notify,audit,database}`.
Single shared connector registry (`connectors/all`) for API + CLI.

## IMPLEMENTED FEATURES (with tests)
- Security hardening (2026-10-02): Redis-backed jti revocation
  (`auth.RedisRevoker`, auto-enabled when Redis answers, memory fallback),
  distributed brute-force counters (`bruteforce` memory+Redis, wired into
  login: 429 lockout, reset on success) — live-tested vs Memurai.
- SNMPv3 privacy: AES-128-CFB per RFC 3826 construction (salt/IV documented
  in code), round-trip + wrong-key + salt-uniqueness tests; agent interop
  still E2E-gated.
- CWMP subset (`internal/cwmp`, implements `tr069.ACS`): Inform ingestion,
  device registry, Get/SetParameterValues + Reboot task queue served as SOAP,
  honest empty-session handling; fake-CPE tests. Full session/auth/file
  transfer PLANNED.
- Frontend: vitest + jsdom + testing-library (4 tests: i18n, Login render);
  `lang.tsx` split so pages import no app side effects; `npm test` + build green.
- Operations domain (2026-10-02): incidents + deterministic correlation
  (child-absorption, severity roll-up), topology impact/paths ("who is
  affected"), read-only digital twin (failure sim, hot links, headroom),
  tickets/work-orders state machine, contracts + SLA breach with maintenance
  exclusions, config snapshots/diffs/approvals/rollback + compliance checks,
  economics (MRR/ARPU/churn/suspension), z-score/threshold anomaly detectors,
  read-only AI copilot (role-gated, evidence-cited, exec-refusing),
  per-subscriber service-health composition, org rate limits, plan
  entitlements, 10-state subscriber lifecycle, live tenant-isolation tests.
- Billing: money-as-cents, invoices/discounts/tax/late-fee, recurring generator
  with proration, dunning state machine, reconciliation (over/under/credit/refund),
  invoice numbering (org-scoped transactional), multi-currency validation + FX,
  balanced double-entry journal, Xendit/Midtrans webhook verification (HMAC).
- Provisioning: 10-step retry/backoff workflow + desired-vs-actual drift detection.
- Jobs: idempotent enqueue, retry/backoff, DLQ, priority, per-device locks,
  per-connector rate buckets, graceful DrainCtx.
- RADIUS: RFC codec, auth/acct server, CoA/Disconnect sender, NAS inventory,
  anti-replay dedup, session store, usage aggregation (no double-count) for billing.
- SNMP: real v2c client (BER codec, GET/WALK/ifTable), vendor profiles
  (MikroTik/Cisco/Generic). v3 USM: PLANNED.
- MikroTik: real RouterOS API + API-SSL + REST; device info/health/interfaces/
  traffic/DHCP/PPP secrets+active/queues/profiles/pools/hotspot/identity/export/reboot;
  provision/suspend/activate/disconnect/delete/update; protocol tests vs scripted
  fake; hardware E2E gated (`MIKROTIK_E2E`).
- IPAM: v4/v6 pools, reservations, O(1) allocation, static claim, VLAN/VRF tags.
- Store: tenant-scoped paginated SQL lists, NOC aggregates, PBKDF2 auth, 503 honesty.
- API: JWT+RBAC, request IDs, error envelope, CORS allowlist, login throttle,
  rate limit, lab test endpoint, topology from inventory. OpenAPI: PARTIAL (update pending).
- Scheduler/workers/ispctl (worker, scheduler, health, doctor, connector list,
  capabilities, device test). Syslog receiver. TR-069 ACS seam. Webhooks HMAC.

## PARTIAL FEATURES
- MikroTik/generic connectors: protocol-implemented, hardware/agent E2E pending.
- Cloud connectors: adapter + capability architecture, credential-gated, fail-closed.
- Router/OLT families: registered, fail-closed, command templates PLANNED.
- Frontend: all menus render with real API data + search/pagination/empty/error
  states; charts, bulk actions, drawers: PLANNED.

## VERIFIED CONNECTORS — NONE
Nothing claims VERIFIED. Evidence-gated promotion only (see matrix evidence fields).

## REQUIRES VENDOR ACCESS — 8 cloud families (Ruijie, Reyee, UniFi, Omada,
## Meraki, Aruba Central, Mist, cnMaestro).

## KNOWN LIMITATIONS (updated 2026-10-02)
- `go test -race` needs cgo/gcc (unavailable on this Windows box; CI/Linux must run it).
- Live PostgreSQL 18.3 verified 2026-10-02: migrations 001–006 applied clean,
  seed + rules present, `TestLivePostgres` passes (`TEST_DATABASE_URL`).
- Live Redis-compatible (Memurai :6379) verified: SET/GET/DEL, distributed
  lock (SET NX PX + Lua release/refresh), Streams (XADD/XGROUP/XREADGROUP/
  XACK + idempotent publish) — `TestLiveRedis/Lock/Stream` pass.
- `ispctl backup` verified end-to-end (63KB pg_dump of live DB).
- Measured micro-benchmarks recorded in PERFORMANCE.md (dev box only).
- Live E2E (MIKROTIK_E2E/RADIUS_E2E/SNMP_E2E) never run — no hardware/agents.
- SNMPv3: authNoPriv (HMAC-MD5/SHA-96, discovery) implemented + round-trip
  tested; privacy (DES/AES) PLANNED; agent interop pending.
- Session revocation (jti denylist + logout) implemented; multi-instance
  Redis backing PLANNED (interface-ready).
- OpenAPI refreshed to v1.1.0 (all current routes).
- HA: stateless API + worker-safe queues by design; Redis Streams transport
  implemented + live-tested; distributed locks live-tested.

## SECURITY STATUS
PBKDF2-SHA256 passwords, AES-GCM credential vault, demo login gated
(`ISP_DEMO_LOGIN=1` + warn), login throttle, per-IP rate limit, CORS allowlist,
SSRF guard, HMAC webhooks, API-key hashing, audit records, no secrets in logs/
responses. Findings: user table needs password-rotation policy (PLANNED);
session revocation list PLANNED.

## SCALABILITY STATUS
Partition-ready accounting/telemetry schema (005), bounded polling concurrency,
per-device locks/rate limits, pagination everywhere, O(1) IPAM, cursor-safe queues.
No benchmark numbers claimed (none measured).

## TEST STATUS
`go test ./...` all green (~25 packages). Protocol tests (MikroTik scripted fake,
SNMP fake agent, RADIUS codec/dedup/usage), domain tests (billing/dunning/ledger/
IPAM/drift/jobs-priority), HTTP tests (auth gate/throttle/503 honesty/request-id).
E2E tests exist but SKIP without env + hardware.

## DEPLOYMENT STATUS — PARTIAL
systemd units, nginx example, env template, `ispctl doctor/health`, migration chain.
Backup/restore procedures documented, not automated. Docker optional (not provided).

## NEXT RISKS (blockers to world-class production)
1. Live PostgreSQL execution of migrations 001–005 + `TEST_DATABASE_URL` CI run.
2. Hardware E2E for MikroTik (needs device) to promote PARTIAL→VERIFIED.
3. SNMP agent interop + RADIUS NAS interop tests.
4. Redis-backed queue transport + distributed locks for multi-worker HA.
5. SNMPv3 USM, full CWMP session server, cloud credential partnerships.
6. Frontend charts/bulk-ops/drawers; OpenAPI refresh; password rotation; session revocation.
