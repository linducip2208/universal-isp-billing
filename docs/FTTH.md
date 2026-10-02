# FTTH

See IPAM.md (combined to avoid drift). Domain tables: `olt_devices`, `onus`
(serial, LOID-ready, rx/tx power, status). OLT families: Huawei MA5600T,
ZTE C320, Nokia 7360, FiberHome AN5116, BDCOM P3310, VSOL V1600, C-Data
FD1608, Dasan V8240, Raisecom ISC, Zyxel OLT1408, Calix E7, ADTRAN TA5000,
DZS Velocity — all PLANNED, fail-closed. ONU provisioning rides the
provisioning workflow with `service_type=ftth` + idempotency keys.
