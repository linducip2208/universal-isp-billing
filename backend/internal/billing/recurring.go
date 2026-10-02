// Recurring billing: generate period invoices from subscriptions with
// mid-cycle proration, discounts, tax, due dates and grace periods.
// Idempotency key = subscriptionID + period, safe to retry/cron.
package billing

import (
	"fmt"
	"time"
)

type Discount struct {
	Kind   string `json:"kind"` // percent_bps | fixed_cents
	Value  int64  `json:"value"`
	Reason string `json:"reason,omitempty"`
}

type RecurringInput struct {
	SubscriptionID string
	CustomerID     string
	PackageName    string
	MonthlyCents   int64
	PeriodStart    time.Time // first day of billed period
	PeriodEnd      time.Time // last day of billed period (inclusive)
	ActiveFrom     time.Time // subscription activation (proration anchor)
	TaxBps         int
	Discounts      []Discount
	DueDays        int
	GraceDays      int
	CreditCents    int64 // carried credit applied first
}

func PeriodKey(subID string, start time.Time) string {
	return fmt.Sprintf("%s:%s", subID, start.UTC().Format("2006-01"))
}

// Generate builds the invoice for one subscription + period.
func Generate(in RecurringInput) *Invoice {
	daysIn := daysInMonth(in.PeriodStart)
	billFrom := in.PeriodStart
	if in.ActiveFrom.After(billFrom) {
		billFrom = in.ActiveFrom
	}
	daysLeft := daysBetween(billFrom, in.PeriodEnd) + 1
	if daysLeft < 0 {
		daysLeft = 0
	}
	if daysLeft > daysIn {
		daysLeft = daysIn
	}
	amount := in.MonthlyCents
	if daysLeft != daysIn {
		amount = int64(Prorate(Money(in.MonthlyCents), daysLeft, daysIn))
	}
	due := in.PeriodEnd.AddDate(0, 0, in.DueDays+1)
	inv := NewInvoice(in.CustomerID, due)
	inv.SubscriptionID = in.SubscriptionID
	label := fmt.Sprintf("%s %s", in.PackageName, in.PeriodStart.Format("Jan 2006"))
	if daysLeft != daysIn {
		label = fmt.Sprintf("%s (prorated %d/%d days)", label, daysLeft, daysIn)
	}
	inv.AddItem(label, 1, Money(amount))
	for _, d := range in.Discounts {
		switch d.Kind {
		case "percent_bps":
			inv.ApplyDiscount(Money(int64(inv.Subtotal) * d.Value / 10000))
		case "fixed_cents":
			inv.ApplyDiscount(Money(d.Value))
		}
	}
	if in.TaxBps > 0 {
		inv.ApplyTaxPct(int(in.TaxBps))
	}
	if in.CreditCents > 0 {
		apply := Money(in.CreditCents)
		if apply > inv.Total {
			apply = inv.Total
		}
		inv.ApplyDiscount(apply)
	}
	if in.GraceDays > 0 {
		g := due.AddDate(0, 0, in.GraceDays)
		inv.GraceUntil = &g
	}
	inv.Status = InvoiceOpen
	return inv
}

func daysInMonth(t time.Time) int {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return int(first.AddDate(0, 1, -1).Day())
}

func daysBetween(a, b time.Time) int {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	da := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	db := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return int(db.Sub(da).Hours() / 24)
}
