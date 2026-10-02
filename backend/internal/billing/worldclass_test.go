package billing_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/billing"
)

func TestRecurringFullMonth(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	inv := billing.Generate(billing.RecurringInput{
		SubscriptionID: "s1", CustomerID: "c1", PackageName: "Home 50M",
		MonthlyCents: 15000000, PeriodStart: start, PeriodEnd: end,
		ActiveFrom: start, TaxBps: 1100, DueDays: 7, GraceDays: 3,
	})
	// 15.000.000 + 11% = 16.650.000
	if inv.Total != 16650000 {
		t.Fatalf("total=%v", inv.Total)
	}
	if inv.Status != billing.InvoiceOpen {
		t.Fatal("should be open")
	}
	if inv.GraceUntil == nil || inv.GraceUntil.Sub(inv.DueAt) != 3*24*time.Hour {
		t.Fatal("grace should be due+3d")
	}
}

func TestRecurringProrated(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	active := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC) // 15 of 31 days
	inv := billing.Generate(billing.RecurringInput{
		SubscriptionID: "s1", CustomerID: "c1", PackageName: "Home 50M",
		MonthlyCents: 3100000, PeriodStart: start, PeriodEnd: end,
		ActiveFrom: active,
	})
	if inv.Total != 1500000 { // 3.100.000 * 15/31
		t.Fatalf("prorated total=%v", inv.Total)
	}
}

func TestRecurringDiscountAndCredit(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	inv := billing.Generate(billing.RecurringInput{
		SubscriptionID: "s1", CustomerID: "c1", PackageName: "Home 50M",
		MonthlyCents: 10000000, PeriodStart: start, PeriodEnd: end, ActiveFrom: start,
		Discounts:   []billing.Discount{{Kind: "percent_bps", Value: 1000}},
		CreditCents: 500000,
	})
	// 10.000.000 - 10% (1.000.000) - 500.000 = 8.500.000
	if inv.Total != 8500000 {
		t.Fatalf("total=%v", inv.Total)
	}
}

func TestDunningStages(t *testing.T) {
	now := time.Now()
	due := now.Add(-10 * 24 * time.Hour)
	grace := now.Add(-8 * 24 * time.Hour)
	inv := billing.NewInvoice("c1", due)
	inv.GraceUntil = &grace
	inv.Status = billing.InvoiceOpen
	pol := billing.DefaultPolicy()
	if got := billing.StageOf(inv, now, pol); got != billing.StageSuspend {
		t.Fatalf("stage=%s want suspend", got)
	}
	old := now.Add(-40 * 24 * time.Hour)
	inv2 := billing.NewInvoice("c1", old)
	inv2.Status = billing.InvoiceOpen
	if got := billing.StageOf(inv2, now, pol); got != billing.StageTerminate {
		t.Fatalf("stage=%s want terminate", got)
	}
	if got := billing.AgingBucket(inv2, now); got != "30+" {
		t.Fatalf("bucket=%s", got)
	}
}

func TestReconcileOverUnder(t *testing.T) {
	inv := billing.NewInvoice("c1", time.Now().Add(24*time.Hour))
	inv.AddItem("Home 50M", 1, 10000000)
	ledger := &billing.CreditLedger{CustomerID: "c1"}
	res, err := billing.Reconcile(inv, ledger, 12000000)
	if err != nil {
		t.Fatal(err)
	}
	if !res.InvoicePaid || res.Overpayment != 2000000 || ledger.Balance != 2000000 {
		t.Fatalf("overpay: %+v ledger=%v", res, ledger.Balance)
	}
	inv2 := billing.NewInvoice("c1", time.Now().Add(24*time.Hour))
	inv2.AddItem("Home 50M", 1, 10000000)
	res2, err := billing.Reconcile(inv2, ledger, 4000000)
	if err != nil {
		t.Fatal(err)
	}
	if res2.InvoicePaid || res2.Underpaid != 6000000 {
		t.Fatalf("underpay: %+v", res2)
	}
	if err := billing.ValidateRefund(10000000, 0, 11000000); err == nil {
		t.Fatal("excess refund must fail")
	}
}
