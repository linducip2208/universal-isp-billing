// Reconciliation: match provider money to invoices, handle over/under
// payment via customer credit, and model refunds. All integer cents.
package billing

import "errors"

type CreditLedger struct {
	CustomerID string `json:"customer_id"`
	Balance    Money  `json:"balance_cents"`
}

func (l *CreditLedger) Add(amount Money) {
	l.Balance += amount
}

func (l *CreditLedger) Take(max Money) Money {
	if l.Balance <= 0 || max <= 0 {
		return 0
	}
	if l.Balance < max {
		max = l.Balance
	}
	l.Balance -= max
	return max
}

type ReconcileResult struct {
	Applied     Money `json:"applied_cents"`
	Overpayment Money `json:"overpayment_cents"`
	Underpaid   Money `json:"underpaid_cents"`
	InvoicePaid bool  `json:"invoice_paid"`
}

// Reconcile applies a verified provider payment to an invoice. Overpayment
// becomes ledger credit; underpayment keeps the invoice open with the
// remaining balance.
func Reconcile(inv *Invoice, ledger *CreditLedger, paid Money) (*ReconcileResult, error) {
	if paid <= 0 {
		return nil, errors.New("paid amount must be positive")
	}
	res := &ReconcileResult{}
	bal := inv.Balance()
	if paid >= bal {
		res.Applied = bal
		res.Overpayment = paid - bal
		if ledger != nil && res.Overpayment > 0 {
			ledger.Add(res.Overpayment)
		}
		_ = inv.ApplyPayment(bal)
		res.InvoicePaid = inv.Status == InvoicePaid
		return res, nil
	}
	res.Applied = paid
	_ = inv.ApplyPayment(paid)
	res.Underpaid = inv.Balance()
	return res, nil
}

type Refund struct {
	PaymentID string `json:"payment_id"`
	Amount    Money  `json:"amount_cents"`
	Reason    string `json:"reason"`
}

// ValidateRefund ensures refunds never exceed what was collected.
func ValidateRefund(collected, refunded, want Money) error {
	if want <= 0 {
		return errors.New("refund must be positive")
	}
	if refunded+want > collected {
		return errors.New("refund exceeds collected amount")
	}
	return nil
}
