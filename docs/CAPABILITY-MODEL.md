# Capability model 2.0

`internal/connectors/sdk`: granular `Capability` IDs (~40: device_info,
device_health, interfaces, traffic, clients, pppoe, ipoe, hotspot, dhcp,
queues, bandwidth, radius, coa, disconnect, provisioning, vlan, olt, onu,
optical_power, tr069, snmp, ap, webhooks, ...), each with `Status`
(VERIFIED|PARTIAL|READY_FOR_CREDENTIALS|REQUIRES_VENDOR_ACCESS|UNSUPPORTED|PLANNED),
`Version`, `MinFirmware`, `Models`, `Note`.

`registry.Entry` carries models/protocols/auth-methods/limitations/
connector-version/evidence; `registry.Descriptors()` serves identity cards to
the API, CLI, and docs. Full dump: `docs/capability-matrix.json`
(`ispctl capabilities`); summary: `docs/capability-matrix.yaml`.

Connection Lab probes each advertised capability live with 15s timeouts and
records declared-vs-observed + evidence. A failed probe never upgrades a
declaration; passing confirms it for that device/model/firmware only.
Mocks are not registered for real families, so lab can never report mock
success for a real device selection.
