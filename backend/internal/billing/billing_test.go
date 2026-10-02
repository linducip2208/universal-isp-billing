package billing_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/billing"
)

func TestMoneyMath(t *testing.T) {
	inv := billing.NewInvoice("c1", time.Now().Add(24*time.Hour))
	inv.AddItem("Home 50M", 1, billing.Money(15000000))
	inv.ApplyDiscount(100000)
	inv.ApplyTaxPct(1100)
	if inv.Total != billing.Money(15000000-100000)+billing.Money((15000000-100000)*1100/10000) {
		t.Fatalf("bad total %v", inv.Total)
	}
	if err := inv.ApplyPayment(inv.Total); err != nil {
		t.Fatal(err)
	}
	if inv.Status != billing.InvoicePaid {
		t.Fatal("should be paid")
	}
}

func TestProrate(t *testing.T) {
	if got := billing.Prorate(3000000, 15, 30); got != 1500000 {
		t.Fatalf("prorate=%v", got)
	}
}

func TestOverdueGrace(t *testing.T) {
	now := time.Now()
	inv := billing.NewInvoice("c1", now.Add(-time.Hour))
	g := now.Add(time.Hour)
	inv.GraceUntil = &g
	if !inv.IsOverdue(now) || !inv.InGrace(now) {
		t.Fatal("should be overdue+in grace")
	}
}
