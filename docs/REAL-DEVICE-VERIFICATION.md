# Real-device verification

Policy: NOTHING is VERIFIED without evidence. Promotion ladder per connector:
AUTOMATED_TESTED (mock/fake transport tests, done for MikroTik/RADIUS/SNMP)
-> LAB_TESTED (Connection Lab probes against a real device, recorded with
firmware/model) -> REAL_DEVICE_VERIFIED (documented E2E run) ->
PRODUCTION_READY (soak + rollback proven).

E2E harnesses (all SKIP without env + hardware, never fail CI):
- MikroTik: `MIKROTIK_E2E=true` + `MIKROTIK_HOST/PORT/USERNAME/PASSWORD/TLS`
  (`internal/connectors/mikrotik` TestE2E: connect, info, interfaces).
- RADIUS: `RADIUS_E2E` (pending harness — NAS interop checklist in CONNECTORS.md).
- SNMP: `SNMP_E2E=1` + `SNMP_TARGET/COMMUNITY` (sysDescr + profile log).

Evidence required for promotion: device model, firmware, date, operator,
lab probe export (`POST /api/v1/lab/test` result JSON), attached to
`docs/connectors/<vendor>.md` + matrix `evidence` field. No exceptions.

Current state (2026-10-02): 0 REAL_DEVICE_VERIFIED, 0 LAB_TESTED, 5 PARTIAL
(protocol-tested), 8 REQUIRES_VENDOR_ACCESS, 25 PLANNED.
