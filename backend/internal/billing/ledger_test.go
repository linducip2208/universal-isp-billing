package billing_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/billing"
)

func TestNumberingFormat(t *testing.T) {
	if got := billing.FormatNumber("DEMO", "202610", 42); got != "INV-DEMO-202610-000042" {
		t.Fatalf("number=%s", got)
	}
}

func TestCurrency(t *testing.T) {
	if err := billing.ValidateCurrency("IDR"); err != nil {
		t.Fatal(err)
	}
	if err := billing.ValidateCurrency("XXX"); err == nil {
		t.Fatal("unknown currency must fail")
	}
	got := billing.Convert(10000, billing.FxRate{From: "USD", To: "IDR", Rate: 16000})
	if got != 160000000 {
		t.Fatalf("fx=%v", got)
	}
}

func TestLedgerBalanced(t *testing.T) {
	e := billing.PostInvoice("o1", "inv1", 10000000, 1100000, "k1")
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	p := billing.PostPayment("o1", "inv1", "pay1", 11100000, "k2")
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := &billing.JournalEntry{Lines: []billing.JournalLine{{Account: "AR", Debit: 5}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("unbalanced must fail")
	}
}
