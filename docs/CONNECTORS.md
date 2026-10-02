# Connectors

Billing calls ONLY `sdk.NetworkConnector`. Vendor logic lives in packages.

- MikroTik (`internal/connectors/mikrotik`): REAL RouterOS API + API-SSL +
  REST implementation (login, sentences, /system/resource, /interface,
  DHCP leases, PPP secrets, queues, hotspot). Mark: VERIFIED (transport) —
  hardware E2E pending, see integration harness.
- Generic (`generic`): RADIUS (CoA/Disconnect), SNMP (read-only monitoring),
  REST/generic-HTTP, SSH/NETCONF/RESTCONF/webhook seams.
- Cloud (`cloud`): Ruijie Cloud, Reyee, UniFi, Omada, Meraki, Aruba Central,
  Mist, cnMaestro — adapter + credential handling + capability declaration,
  marked REQUIRES_VENDOR_ACCESS (no endpoints fabricated).
- Routers/OLT: Cisco, Juniper, Huawei, ZTE, Nokia, VyOS, Fortinet, EdgeRouter,
  Aruba CX + 13 OLT families via generic transports, MODEL_DEPENDENT.

Connection Lab: TestConnection -> GetDeviceInfo -> GetInterfaces ->
GetCapabilities, persisted to devices/device_capabilities + audit.
