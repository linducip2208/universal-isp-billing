// Package billing implements money-safe billing primitives.
// All money is int64 minor units (cents). Never float64.
package billing

import (
	"errors"
	"fmt"
	"time"
)

type Money int64

func (m Money) Add(o Money) Money { return m + o }
func (m Money) Mul(qty int) Money { return m * Money(qty) }
func (m Money) String() string {
	neg := m < 0
	if neg {
		m = -m
	}
	s := fmt.Sprintf("%d.%02d", int64(m)/100, int64(m)%100)
	if neg {
		return "-" + s
	}
	return s
}

type InvoiceStatus string

const (
	InvoiceDraft   InvoiceStatus = "draft"
	InvoiceOpen    InvoiceStatus = "open"
	InvoicePaid    InvoiceStatus = "paid"
	InvoiceVoid    InvoiceStatus = "void"
	InvoiceOverdue InvoiceStatus = "overdue"
)

type InvoiceItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitAmount  Money  `json:"unit_amount"`
	Amount      Money  `json:"amount"`
}

type Invoice struct {
	ID             string        `json:"id"`
	CustomerID     string        `json:"customer_id"`
	SubscriptionID string        `json:"subscription_id,omitempty"`
	Items          []InvoiceItem `json:"items"`
	Subtotal       Money         `json:"subtotal"`
	Discount       Money         `json:"discount"`
	Tax            Money         `json:"tax"`
	LateFee        Money         `json:"late_fee"`
	Total          Money         `json:"total"`
	AmountPaid     Money         `json:"amount_paid"`
	Status         InvoiceStatus `json:"status"`
	DueAt          time.Time     `json:"due_at"`
	GraceUntil     *time.Time    `json:"grace_until,omitempty"`
}

func NewInvoice(customerID string, dueAt time.Time) *Invoice {
	return &Invoice{CustomerID: customerID, Status: InvoiceDraft, DueAt: dueAt}
}

func (in *Invoice) AddItem(desc string, qty int, unit Money) {
	amt := unit.Mul(qty)
	in.Items = append(in.Items, InvoiceItem{Description: desc, Quantity: qty, UnitAmount: unit, Amount: amt})
	in.recalc()
}

func (in *Invoice) recalc() {
	var sub Money
	for _, it := range in.Items {
		sub += it.Amount
	}
	in.Subtotal = sub
	in.Total = sub - in.Discount + in.Tax + in.LateFee
	if in.Total < 0 {
		in.Total = 0
	}
}

func (in *Invoice) ApplyDiscount(amount Money) {
	in.Discount += amount
	in.recalc()
}

// ApplyTaxPct applies integer-basis-point tax, e.g. 1100 = 11%.
func (in *Invoice) ApplyTaxPct(bps int) {
	base := int64(in.Subtotal - in.Discount)
	if base < 0 {
		base = 0
	}
	in.Tax = Money(base * int64(bps) / 10000)
	in.recalc()
}

func (in *Invoice) ApplyLateFee(amount Money) {
	in.LateFee += amount
	in.recalc()
}

func (in *Invoice) ApplyPayment(amount Money) error {
	if amount <= 0 {
		return errors.New("payment must be positive")
	}
	in.AmountPaid += amount
	if in.AmountPaid >= in.Total {
		in.Status = InvoicePaid
	} else if in.Status == InvoiceDraft {
		in.Status = InvoiceOpen
	}
	return nil
}

func (in *Invoice) Balance() Money { return in.Total - in.AmountPaid }

func (in *Invoice) IsOverdue(now time.Time) bool {
	if in.Status == InvoicePaid || in.Status == InvoiceVoid {
		return false
	}
	return now.After(in.DueAt)
}

func (in *Invoice) InGrace(now time.Time) bool {
	if in.GraceUntil == nil {
		return false
	}
	return now.After(in.DueAt) && !now.After(*in.GraceUntil)
}

// Prorate computes minor-unit proration: monthly * daysLeft/daysInMonth.
func Prorate(monthly Money, daysLeft, daysInMonth int) Money {
	if daysInMonth <= 0 || daysLeft <= 0 {
		return 0
	}
	if daysLeft > daysInMonth {
		daysLeft = daysInMonth
	}
	return Money(int64(monthly) * int64(daysLeft) / int64(daysInMonth))
}
