# Billing

Money = int64 cents. Modules: `billing.go` (invoice math), `recurring.go`
(period generation + proration + discounts + tax + credits), `dunning.go`
(current→grace→reminder→warning→suspend→terminate + aging buckets),
`reconcile.go` (over/under-payment, credit ledger, refund guards),
`numbering.go` (org-scoped transactional `INV-ORG-YYYYMM-######`, 12
currencies + dated FX), `ledger.go` (balanced double-entry journal with
idempotency), `provider.go` + `gateways.go` (Manual/Xendit/Midtrans with real
webhook verification, fail-closed without keys).

Flow: Invoice → Payment → webhook verify → reconcile → activate →
provisioning job (idempotent) → connector → device. Overdue: dunning stages
→ suspend job (CoA/Disconnect or API) → reactivate on pay. Network failure
never corrupts billing state (explicit stage machines + DLQ).
