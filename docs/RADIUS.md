# RADIUS

Native Go: Access-Request/Accept/Reject, Accounting Start/Interim/Stop,
CoA/Disconnect (RFC 5176), NAS inventory (`NASStore`, per-IP shared secrets
via vault refs), anti-replay dedup (60s window, cached responses),
session tracking, usage aggregation (`UsageStore`, forward-delta only — no
double billing on retransmit) feeding billing/analytics, flexible vendor
dictionary (MikroTik/Cisco/Huawei/ZTE). IPv4/IPv6 framed addresses.
Run: `RADIUS_AUTH_PORT/ACCT_PORT` (1812/1813). High volume: partition
`radius_accounting` monthly (005). Tests: codec, sessions, NAS auth, dedup,
usage. Interop E2E: `RADIUS_E2E` (pending).
