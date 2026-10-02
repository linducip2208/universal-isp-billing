package billing

import (
	"errors"
	"sync"
)

// ErrDuplicateInvoice is returned when the same subscription+period is
// generated twice. Combined with the SQL sequence and a UNIQUE period-key
// constraint, double-generation is impossible through this issuer.
var ErrDuplicateInvoice = errors.New("invoice already issued for subscription+period")

// Issuer guards period-invoice generation against duplicates (in-process;
// the SQL sequence + a UNIQUE(period key) constraint guard across instances).
type Issuer struct {
	mu   sync.Mutex
	seen map[string]string // periodKey -> invoiceID
}

func NewIssuer() *Issuer { return &Issuer{seen: map[string]string{}} }

// Issue generates exactly once per (subscription, period). The second call
// with the same key returns the original invoice ID and ErrDuplicateInvoice.
func (iss *Issuer) Issue(in RecurringInput) (*Invoice, error) {
	key := PeriodKey(in.SubscriptionID, in.PeriodStart)
	iss.mu.Lock()
	defer iss.mu.Unlock()
	if id, ok := iss.seen[key]; ok {
		return nil, &DuplicateError{Key: key, InvoiceID: id}
	}
	inv := Generate(in)
	if inv.ID == "" {
		inv.ID = "inv-" + key
	}
	iss.seen[key] = inv.ID
	return inv, nil
}

type DuplicateError struct {
	Key       string
	InvoiceID string
}

func (e *DuplicateError) Error() string        { return "duplicate invoice for " + e.Key }
func (e *DuplicateError) Is(target error) bool { return target == ErrDuplicateInvoice }
