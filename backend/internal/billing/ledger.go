package billing

import "errors"

// Ledger: accounting-ready double-entry journal. Every posting balances
// (sum debit == sum credit) and carries an idempotency key so retries and
// duplicate webhooks never double-post.
type JournalLine struct {
	Account string `json:"account"` // e.g. AR, REVENUE, CASH, TAX_PAYABLE, CREDIT
	Debit   Money  `json:"debit_cents"`
	Credit  Money  `json:"credit_cents"`
}

type JournalEntry struct {
	ID             string        `json:"id"`
	OrgID          string        `json:"org_id"`
	InvoiceID      string        `json:"invoice_id,omitempty"`
	PaymentID      string        `json:"payment_id,omitempty"`
	Memo           string        `json:"memo"`
	Lines          []JournalLine `json:"lines"`
	IdempotencyKey string        `json:"idempotency_key"`
}

func (e *JournalEntry) Validate() error {
	if len(e.Lines) < 2 {
		return errors.New("journal needs at least 2 lines")
	}
	var dr, cr Money
	for _, l := range e.Lines {
		if l.Debit < 0 || l.Credit < 0 {
			return errors.New("negative journal amount")
		}
		if l.Debit > 0 && l.Credit > 0 {
			return errors.New("line cannot be both debit and credit")
		}
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr {
		return errors.New("unbalanced journal")
	}
	if dr == 0 {
		return errors.New("zero journal")
	}
	return nil
}

// PostInvoice builds the accrual entry: Dr AR / Cr REVENUE / Cr TAX_PAYABLE.
func PostInvoice(orgID, invoiceID string, subtotal, tax Money, key string) *JournalEntry {
	lines := []JournalLine{
		{Account: "AR", Debit: subtotal + tax},
		{Account: "REVENUE", Credit: subtotal},
	}
	if tax > 0 {
		lines = append(lines, JournalLine{Account: "TAX_PAYABLE", Credit: tax})
	}
	return &JournalEntry{OrgID: orgID, InvoiceID: invoiceID, Memo: "invoice issued", Lines: lines, IdempotencyKey: key}
}

// PostPayment builds the cash entry: Dr CASH / Cr AR.
func PostPayment(orgID, invoiceID, paymentID string, amount Money, key string) *JournalEntry {
	return &JournalEntry{OrgID: orgID, InvoiceID: invoiceID, PaymentID: paymentID, Memo: "payment collected",
		Lines:          []JournalLine{{Account: "CASH", Debit: amount}, {Account: "AR", Credit: amount}},
		IdempotencyKey: key}
}
