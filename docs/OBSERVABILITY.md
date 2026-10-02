# Observability / HA / Performance

- Logs: JSON slog, request IDs, no secrets. Metrics: `/metrics` (Prometheus
  exposition stub counters — full instrumentation PLANNED). Health:
  `/health /ready /live` (+ `ispctl health`).
- HA design (documented, not deployed): stateless API behind LB, N workers
  with Redis transport + distributed locks, Postgres replication-ready,
  graceful shutdown everywhere, job dedup via idempotency keys.
- Performance: partition-ready high-volume tables (005), per-device locks,
  connector rate buckets, bounded poll concurrency, pagination, O(1) IPAM.
  No benchmarks claimed. `go test -race` must run in CI (needs gcc).
