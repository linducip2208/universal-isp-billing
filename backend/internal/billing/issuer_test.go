package billing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/billing"
)

func TestIssuerOnce(t *testing.T) {
	iss := billing.NewIssuer()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	in := billing.RecurringInput{SubscriptionID: "s9", CustomerID: "c9", PackageName: "P",
		MonthlyCents: 1000000, PeriodStart: start, PeriodEnd: end, ActiveFrom: start}
	inv, err := iss.Issue(in)
	if err != nil || inv.ID == "" {
		t.Fatalf("first issue: %+v %v", inv, err)
	}
	if _, err := iss.Issue(in); !errors.Is(err, billing.ErrDuplicateInvoice) {
		t.Fatalf("second issue must ErrDuplicateInvoice, got %v", err)
	}
	var dup *billing.DuplicateError
	if _, err := iss.Issue(in); !errors.As(err, &dup) || dup.InvoiceID != inv.ID {
		t.Fatalf("dup must reference original: %v", err)
	}
}
