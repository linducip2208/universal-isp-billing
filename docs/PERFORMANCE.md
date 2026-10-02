# Performance

See OBSERVABILITY.md (combined). Targets: 100k+ subs, 10k+ devices, millions
of accounting rows. Techniques: partitioning, indexes (004), bounded
concurrency, rate buckets, pagination, cursor allocation. Load-test harness:
PLANNED (no fabricated numbers).

## Measured micro-benchmarks (dev box, Intel Ultra 7 155H, Windows, 2026-10-02)
`go test -bench` (100 iterations, `bench_test.go` in each package):

- billing.Generate (recurring invoice + proration + tax): ~1212 ns/op
- billing.Reconcile (payment apply + ledger): ~714 ns/op
- ipam.AllocateNext (cursor allocation /24+): ~388 ns/op
- radius codec round-trip (encode+decode Access-Request): ~890 ns/op

These are single-function timings on one machine, NOT production throughput
claims. Re-run with `go test -bench . ./internal/...` on target hardware.
