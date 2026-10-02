# Scale

Targets: 100k+ subs, 10k+ devices, 100+ families, millions of
accounting/telemetry rows. Proven on live PostgreSQL 18.3 (2026-10-02):
composite `(org_id, created_at)` indexes (009) turn list endpoints from
Seq Scan into Bitmap Index Scan (EXPLAIN recorded in history); monthly
partition template for `radius_accounting` (005); measured micro-benchmarks
(dev box Ultra 7 155H): billing.Generate ~1212ns, Reconcile ~714ns,
IPAM alloc ~388ns, RADIUS codec ~890ns, queue drain ~52µs/1000 jobs.
Multi-instance: stateless API, Redis Streams transport + distributed locks
both live-tested. Not yet load-tested at target volumes — no throughput
claims beyond the measured numbers above.
