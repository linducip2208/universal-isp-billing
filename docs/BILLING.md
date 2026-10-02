# Billing

Money = int64 cents. Invoices, items, discounts, basis-point tax, late fees,
partial payments, overdue/grace. Providers via `PaymentProvider`
(manual/bank/VA/QRIS/e-wallet/card; Indonesian gateways plug in).
Flow: Invoice -> Payment -> webhook verify -> activate -> provisioning job.
Overdue: grace -> warn -> suspend job (CoA/Disconnect or API) -> reactivate on pay.
