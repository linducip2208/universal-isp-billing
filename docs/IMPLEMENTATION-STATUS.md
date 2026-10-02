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

## KNOWN LIMITATIONS
- `go test -race` needs cgo/gcc (unavailable on this Windows box; CI/Linux must run it).
- No live PostgreSQL/Redis here: DB integration tests gate on `TEST_DATABASE_URL`;
  migrations are carefully written but NOT executed against a live server yet.
- Live E2E (MIKROTIK_E2E/RADIUS_E2E/SNMP_E2E) never run — no hardware/agents.
- OpenAPI file lags new endpoints (`/lab/test`, `/topology`, `/audit`).
- HA: stateless API + worker-safe queues by design; Redis Streams transport,
  distributed locks, and multi-instance deployment are documented, not deployed.

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
