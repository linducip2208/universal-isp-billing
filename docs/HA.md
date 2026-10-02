# HA

See OBSERVABILITY.md (combined). Current capability: single API + worker
processes, shared Postgres, optional Redis. Multi-instance: architecture-ready
(stateless handlers, idempotent jobs, per-device locks need Redis rollout for
cross-process mutual exclusion — PLANNED).
