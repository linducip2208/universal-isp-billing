# Connectors

Billing calls ONLY `sdk.NetworkConnector`. One shared registry
(`internal/connectors/all`) serves API + CLI + docs.

- MikroTik: REAL RouterOS API + API-SSL + REST (login, sentences,
  resource/interfaces/DHCP/PPP secrets+active/queues/profiles/pools/hotspot/
  identity/export/reboot, provision/suspend/activate/disconnect/delete).
  Status: PARTIAL — protocol-tested (scripted fake), hardware E2E pending
  (`MIKROTIK_E2E`). Nothing is VERIFIED anywhere in this repo.
- Generic: RADIUS (CoA/Disconnect, PARTIAL), SNMP v2c client + profiles
  (PARTIAL, fake-agent tested, `SNMP_E2E` pending), REST/generic-HTTP (PARTIAL).
- Cloud (8 families): adapter + credential handling + capability declarations,
  REQUIRES_VENDOR_ACCESS, fail-closed (mutations error without API session).
- Routers (12) / OLT (13): registered, PLANNED, fail-closed — no silent success.

Connection Lab probes every advertised capability live and records
declared-vs-observed. See CAPABILITY-MODEL.md, CONNECTION-LAB.md.
