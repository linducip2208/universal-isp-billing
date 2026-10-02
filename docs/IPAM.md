# IPAM / FTTH / Network automation

- IPAM (`internal/ipam`): v4/v6 prefixes, pools, reservations, static claims,
  O(1) cursor allocation, VLAN/VRF tags, utilization. CGNAT pools supported
  as ordinary prefixes with metadata.
- FTTH (`internal/connectors/olt` + `olt_devices`/`onus` tables + TR-069 seam):
  OLT/PON/ONU modeling, optical fields (rx/tx/distance), 13 vendor families
  registered as PLANNED fail-closed adapters. Full CWMP session server: PLANNED.
- Automation (`internal/automation` + rules seed 003): event-driven
  triggers→actions, observable + auditable executions; drift plans feed it.
