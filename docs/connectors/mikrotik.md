# MikroTik connector

Transport: RouterOS API (8728), API-SSL (8729), REST (`/rest`).
Auth: username/password. Test: `/system/resource/print`.
Ops: device info/health, interfaces, traffic, DHCP leases, PPP secrets/active,
queues, PPP profiles, IP pools, hotspot users, identity, /export backup,
/system/reboot (audited), provision/suspend/activate/disconnect/delete/update.
Status: PARTIAL — protocol tests vs scripted fake (`mikrotik_test.go`);
hardware E2E via `MIKROTIK_E2E=true` + `MIKROTIK_HOST/PORT/USERNAME/PASSWORD/TLS`
(never commit credentials). Promotion to VERIFIED requires a passing E2E run.
