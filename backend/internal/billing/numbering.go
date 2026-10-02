package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Invoice numbering: INV-{ORG}-{YYYYMM}-{SEQ}. Sequence is org-scoped and
// allocated inside a transaction (SELECT ... FOR UPDATE) so concurrent
// billing runs never duplicate numbers.
type NumberSequence interface {
	Next(ctx context.Context, org, period string) (int64, error)
}

func FormatNumber(orgCode, period string, seq int64) string {
	return fmt.Sprintf("INV-%s-%s-%06d", orgCode, period, seq)
}

// SQLSequence implements NumberSequence over invoice_sequences.
type SQLSequence struct{ db *sql.DB }

func NewSQLSequence(db *sql.DB) *SQLSequence { return &SQLSequence{db: db} }

func (s *SQLSequence) Next(ctx context.Context, org, period string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var seq int64
	err = tx.QueryRowContext(ctx,
		`SELECT seq FROM invoice_sequences WHERE org_id=$1 AND period=$2 FOR UPDATE`, org, period).Scan(&seq)
	if err == sql.ErrNoRows {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO invoice_sequences(org_id, period, seq) VALUES($1,$2,1)`, org, period); err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE invoice_sequences SET seq=seq+1 WHERE org_id=$1 AND period=$2`, org, period); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return seq + 1, nil
}

// Currency: ISO-4217 code carried on invoices/payments. Amounts remain
// integer minor units in that currency. Conversion uses explicit dated rates.
type FxRate struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	Rate float64   `json:"rate"`
	At   time.Time `json:"at"`
}

var supportedCurrencies = map[string]bool{
	"IDR": true, "USD": true, "EUR": true, "SGD": true, "MYR": true,
	"THB": true, "PHP": true, "VND": true, "JPY": true, "AUD": true,
	"GBP": true, "CNY": true,
}

func ValidateCurrency(code string) error {
	if !supportedCurrencies[code] {
		return fmt.Errorf("unsupported currency %q", code)
	}
	return nil
}

// Convert minor units using a dated rate (rounds half away from zero).
func Convert(amount Money, rate FxRate) Money {
	if rate.Rate <= 0 {
		return 0
	}
	v := float64(amount) * rate.Rate
	if v >= 0 {
		return Money(v + 0.5)
	}
	return Money(v - 0.5)
}

var ErrSeqNoDB = errors.New("no database for numbering")
