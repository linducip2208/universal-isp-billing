package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/universal-isp/platform/internal/billing"
	"github.com/universal-isp/platform/internal/customers"
	"github.com/universal-isp/platform/internal/security"
)

// Write paths. Server computes all money/totals — client-supplied amounts
// are line items only, never trusted totals. Every write is tenant-scoped.

func (s *Store) CreateCustomer(ctx context.Context, orgID string, c customers.Customer) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	c.OrgID = orgID
	c.Status = customers.StatusActive
	if err := customers.Validate(&c); err != nil {
		return "", err
	}
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO customers(org_id,name,email,phone,status) VALUES($1,$2,NULLIF($3,''),NULLIF($4,''),$5) RETURNING id`,
		orgID, c.Name, c.Email, c.Phone, string(c.Status)).Scan(&id)
	return id, err
}

func (s *Store) CreateSubscription(ctx context.Context, orgID, customerID, packageID, username, service string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	if username == "" {
		return "", fmt.Errorf("username required")
	}
	switch service {
	case "pppoe", "ipoe", "hotspot", "ftth":
	default:
		return "", fmt.Errorf("unknown service %q", service)
	}
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO subscriptions(org_id,customer_id,package_id,username,service,status)
		 VALUES($1,$2,$3,$4,$5,'pending') RETURNING id`,
		orgID, customerID, packageID, username, service).Scan(&id)
	return id, err
}

type InvoiceLine struct {
	Description string `json:"description"`
	Qty         int    `json:"qty"`
	UnitCents   int64  `json:"unit_cents"`
}

// CreateInvoice builds totals server-side and assigns an org-scoped number.
// Idempotency: callers pass idemKey (UNIQUE per org via journal-style guard
// table invoice_idem).
func (s *Store) CreateInvoice(ctx context.Context, orgID, customerID, subscriptionID string, lines []InvoiceLine, taxBps int64, dueAt time.Time, graceDays int, idemKey string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("at least one line required")
	}
	inv := billing.NewInvoice(customerID, dueAt)
	for _, l := range lines {
		if l.Qty <= 0 || l.UnitCents < 0 {
			return "", fmt.Errorf("bad line %q", l.Description)
		}
		inv.AddItem(l.Description, l.Qty, billing.Money(l.UnitCents))
	}
	if taxBps > 0 {
		inv.ApplyTaxPct(int(taxBps))
	}
	period := dueAt.UTC().Format("200601")
	seq := billing.NewSQLSequence(s.db)
	n, err := seq.Next(ctx, orgID, period)
	if err != nil {
		return "", err
	}
	number := billing.FormatNumber("ISP", period, n)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if idemKey != "" {
		var dup string
		err := tx.QueryRowContext(ctx, `SELECT invoice_id FROM invoice_idem WHERE org_id=$1 AND idem_key=$2`, orgID, idemKey).Scan(&dup)
		if err == nil {
			return dup, billing.ErrDuplicateInvoice
		}
	}
	var id string
	grace := dueAt.AddDate(0, 0, graceDays)
	err = tx.QueryRowContext(ctx, `INSERT INTO invoices
		(org_id,customer_id,subscription_id,subtotal_cents,discount_cents,tax_cents,late_fee_cents,total_cents,status,due_at,grace_until,number)
		VALUES($1,$2,NULLIF($3,'')::uuid,$4,0,$5,0,$6,'open',$7,$8,$9) RETURNING id`,
		orgID, customerID, subscriptionID, int64(inv.Subtotal), int64(inv.Tax), int64(inv.Total), dueAt, grace, number).Scan(&id)
	if err != nil {
		return "", err
	}
	for _, l := range lines {
		if _, err := tx.ExecContext(ctx, `INSERT INTO invoice_items(invoice_id,description,qty,unit_cents,amount_cents)
			VALUES($1,$2,$3,$4,$5)`, id, l.Description, l.Qty, l.UnitCents, int64(l.UnitCents)*int64(l.Qty)); err != nil {
			return "", err
		}
	}
	if idemKey != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO invoice_idem(org_id,idem_key,invoice_id) VALUES($1,$2,$3)`, orgID, idemKey, id); err != nil {
			return "", err
		}
	}
	return id, tx.Commit()
}

// RecordPayment inserts a payment and applies it to the invoice atomically.
// Idempotent on idemKey (UNIQUE): repeats return the original payment.
func (s *Store) RecordPayment(ctx context.Context, orgID, invoiceID string, amount int64, method, provider, reference, idemKey string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if idemKey != "" {
		var dup string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM payments WHERE idempotency_key=$1`, idemKey).Scan(&dup); err == nil {
			return dup, nil
		}
	}
	var total, paid int64
	var status string
	err = tx.QueryRowContext(ctx, `SELECT total_cents, paid_cents, status FROM invoices WHERE org_id=$1 AND id=$2 FOR UPDATE`,
		orgID, invoiceID).Scan(&total, &paid, &status)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invoice not found")
	}
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRowContext(ctx, `INSERT INTO payments(org_id,invoice_id,amount_cents,method,provider,reference,status,idempotency_key)
		VALUES($1,$2,$3,$4,$5,$6,'paid',NULLIF($7,'')) RETURNING id`,
		orgID, invoiceID, amount, method, provider, reference, idemKey).Scan(&id)
	if err != nil {
		return "", err
	}
	paid += amount
	newStatus := "open"
	if paid >= total {
		newStatus = "paid"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invoices SET paid_cents=$3, status=$4 WHERE id=$1 AND org_id=$2`, invoiceID, orgID, paid, newStatus); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

// ApplyWebhook settles a provider callback: finds the payment by
// (provider, reference), marks paid exactly once, and applies to invoice.
func (s *Store) ApplyWebhook(ctx context.Context, orgID, provider, reference string, paid bool) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var payID, invoiceID, status string
	var amount int64
	err = tx.QueryRowContext(ctx, `SELECT id, invoice_id, status, amount_cents FROM payments
		WHERE org_id=$1 AND provider=$2 AND reference=$3 FOR UPDATE`, orgID, provider, reference).Scan(&payID, &invoiceID, &status, &amount)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("unknown payment reference")
	}
	if err != nil {
		return "", err
	}
	if status == "paid" {
		return payID, nil // already applied: idempotent replay
	}
	if !paid {
		_, err = tx.ExecContext(ctx, `UPDATE payments SET status='failed' WHERE id=$1`, payID)
		if err != nil {
			return "", err
		}
		return payID, tx.Commit()
	}
	var total, paidCents int64
	if err := tx.QueryRowContext(ctx, `SELECT total_cents, paid_cents FROM invoices WHERE id=$1 AND org_id=$2 FOR UPDATE`,
		invoiceID, orgID).Scan(&total, &paidCents); err != nil {
		return "", err
	}
	paidCents += amount
	newStatus := "open"
	if paidCents >= total {
		newStatus = "paid"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payments SET status='paid' WHERE id=$1`, payID); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invoices SET paid_cents=$3, status=$4 WHERE id=$1 AND org_id=$2`,
		invoiceID, orgID, paidCents, newStatus); err != nil {
		return "", err
	}
	return payID, tx.Commit()
}

// LookupPaymentOrg resolves the tenant from a provider reference so public
// webhooks can be scoped without authentication (signature still required).
func (s *Store) LookupPaymentOrg(ctx context.Context, provider, reference string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	var org string
	err := s.db.QueryRowContext(ctx, `SELECT org_id FROM payments WHERE provider=$1 AND reference=$2`,
		provider, reference).Scan(&org)
	return org, err
}

// CreateVoucher inserts a generated code (uniqueness enforced per org).
func (s *Store) CreateVoucher(ctx context.Context, orgID, code, packageID string, hours int, expires *time.Time) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	if code == "" || hours <= 0 {
		return "", fmt.Errorf("code and positive duration required")
	}
	var id string
	err := s.db.QueryRowContext(ctx, `INSERT INTO vouchers(org_id,code,package_id,duration_hours,expires_at)
		VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5) RETURNING id`,
		orgID, code, packageID, hours, expires).Scan(&id)
	return id, err
}

// RedeemVoucher consumes a code exactly once (guarded transition: only
// active + unexpired rows flip; concurrent redeems get 0 rows).
func (s *Store) RedeemVoucher(ctx context.Context, orgID, code, usedBy string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	res, err := s.db.ExecContext(ctx, `UPDATE vouchers SET status='used', used_by=$3, used_at=now()
		WHERE org_id=$1 AND code=$2 AND status='active' AND (expires_at IS NULL OR expires_at > now())`,
		orgID, code, usedBy)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("voucher invalid, used, or expired")
	}
	return nil
}

// CreateRadiusUser provisions RADIUS credentials (PBKDF2 hash at rest).
func (s *Store) CreateRadiusUser(ctx context.Context, orgID, username, password, profileID string, maxSessions int) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	if username == "" {
		return "", fmt.Errorf("username required")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return "", err
	}
	if maxSessions <= 0 {
		maxSessions = 1
	}
	var id string
	err = s.db.QueryRowContext(ctx, `INSERT INTO radius_users(org_id,username,password_hash,profile_id,max_sessions)
		VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5) RETURNING id`,
		orgID, username, hash, profileID, maxSessions).Scan(&id)
	return id, err
}

// DisableRadiusUser expires credentials immediately (revocation).
func (s *Store) DisableRadiusUser(ctx context.Context, orgID, username string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	res, err := s.db.ExecContext(ctx, `UPDATE radius_users SET expires_at=now() WHERE org_id=$1 AND username=$2`, orgID, username)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("radius user not found")
	}
	return nil
}

// ChangePassword enforces policy (min length via hasher) and rotates hash.
func (s *Store) ChangePassword(ctx context.Context, userID, old, new string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	var hash string
	if err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&hash); err != nil {
		return err
	}
	if err := security.VerifyPassword(old, hash); err != nil {
		return fmt.Errorf("current password incorrect")
	}
	nh, err := security.HashPassword(new)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET password_hash=$2, password_changed_at=now() WHERE id=$1`, userID, nh)
	return err
}

// ChangePasswordByUsername resolves the user inside the org first (IDOR-safe:
// usernames are org-scoped, never global).
func (s *Store) ChangePasswordByUsername(ctx context.Context, orgID, username, old, new string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	var id string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE org_id=$1 AND username=$2 AND deleted_at IS NULL`,
		orgID, username).Scan(&id); err != nil {
		return fmt.Errorf("user not found")
	}
	return s.ChangePassword(ctx, id, old, new)
}

// VerifyAPIKey authenticates X-API-Key (SHA-256 of raw key) and returns org + scopes.
func (s *Store) VerifyAPIKey(ctx context.Context, raw string) (keyID, orgID string, scopes []string, err error) {
	if s == nil || s.db == nil {
		return "", "", nil, ErrNoDatabase
	}
	sum := sha256.Sum256([]byte(raw))
	err = s.db.QueryRowContext(ctx, `SELECT id, org_id, scopes FROM api_keys
		WHERE key_hash=$1 AND revoked=false`, hex.EncodeToString(sum[:])).Scan(&keyID, &orgID, pq.Array(&scopes))
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil, fmt.Errorf("invalid api key")
		}
		return "", "", nil, err
	}
	return keyID, orgID, scopes, nil
}
