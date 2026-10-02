# MikroTik connector

Transport: RouterOS API (8728), API-SSL (8729), REST (`/rest`).
Auth: username/password. Test: `/system/resource/print`.
Capabilities: PPPoE, Hotspot, RADIUS, Queue, DHCP, Firewall, interface
monitoring, disconnect, provisioning — VERIFIED at transport level via
mock-transport tests; hardware E2E via `ispctl device test` (set
MT_HOST/MT_USER/MT_PASS). Integration tests: `mikrotik_test.go`.
