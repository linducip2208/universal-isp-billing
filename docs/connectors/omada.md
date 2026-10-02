# Omada / Meraki / Aruba Central / Mist / cnMaestro

Each has a dedicated package constructor in `internal/connectors/cloud`
with its documented base URL, credential handling, and capability matrix.
All are REQUIRES_VENDOR_ACCESS until vendor credentials + API tests exist.
Generic fallback: use Generic REST/SNMP/RADIUS connectors for monitoring
and CoA today.
