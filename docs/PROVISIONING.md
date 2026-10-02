# Provisioning

10-step workflow (validate sub/device, discover caps, select connector, plan,
execute, verify, record, audit, event). Retry + exponential backoff, 30s step
timeout, DLQ after 5 attempts, idempotency keys end-to-end.
