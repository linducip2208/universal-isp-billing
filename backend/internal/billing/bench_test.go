package billing_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/billing"
)

func BenchmarkGenerate(b *testing.B) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	in := billing.RecurringInput{SubscriptionID: "s", CustomerID: "c", PackageName: "P",
		MonthlyCents: 15000000, PeriodStart: start, PeriodEnd: end, ActiveFrom: start,
		TaxBps: 1100, DueDays: 7, GraceDays: 3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = billing.Generate(in)
	}
}

func BenchmarkReconcile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		inv := billing.NewInvoice("c", time.Now().Add(24*time.Hour))
		inv.AddItem("P", 1, 15000000)
		ledger := &billing.CreditLedger{CustomerID: "c"}
		_, _ = billing.Reconcile(inv, ledger, 15000000)
	}
}
